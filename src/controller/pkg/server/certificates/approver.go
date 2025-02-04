// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package certificates

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/controller/pkg/client"
	"github.com/supernetes/supernetes/controller/pkg/environment"
	certv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cgocertv1 "k8s.io/client-go/kubernetes/typed/certificates/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: Watcher and auto-approver controller for Supernetes node CSRs, since kubelet-serving-cert-approver disallows
//  `system:node:...` CSRs from being sent by the controller

type approver struct {
	csrInterface   cgocertv1.CertificateSigningRequestInterface
	serviceAccount string
}

func (a *approver) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	csr, err := a.csrInterface.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return ctrl.Result{}, crclient.IgnoreNotFound(err)
	}

	// Check if this is a Supernetes CSR
	if csr.Spec.SignerName != certv1.KubeletServingSignerName {
		log.Trace().Msg("unknown signer, skipping")
		return ctrl.Result{}, nil
	}

	if csr.Spec.Username != a.serviceAccount {
		log.Trace().Msg("not requested by Supernetes SA, skipping")
		return ctrl.Result{}, nil
	}

	// Check if the CSR is already approved
	if isApproved(csr) {
		log.Trace().Msg("CSR already approved, skipping")
		return ctrl.Result{}, nil
	}

	approve(csr)
	_, err = a.csrInterface.UpdateApproval(ctx, csr.Name, csr, metav1.UpdateOptions{})
	return ctrl.Result{}, err
}

func approve(csr *certv1.CertificateSigningRequest) {
	csr.Status.Conditions = append(csr.Status.Conditions, certv1.CertificateSigningRequestCondition{
		Type:           certv1.CertificateApproved,
		Status:         corev1.ConditionTrue,
		Reason:         "Approved by Supernetes Controller",
		LastUpdateTime: metav1.Time{Time: time.Now().UTC()},
		Message:        "Auto-approving Supernetes node Kubelet serving certificate",
	})
}

func isApproved(csr *certv1.CertificateSigningRequest) bool {
	for _, condition := range csr.Status.Conditions {
		if condition.Type == certv1.CertificateApproved {
			return true
		}
	}
	return false
}

func Run(ctx context.Context, kubeConfig *rest.Config, env environment.Environment) error {
	serviceAccount, err := getServiceAccount(env)
	if err != nil {
		return fmt.Errorf("failed to retrieve service account: %w", err)
	}

	mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{})
	if err != nil {
		return err
	}

	kubeClient, err := client.NewKubeClient(kubeConfig)
	if err != nil {
		return err
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&certv1.CertificateSigningRequest{}).
		Complete(&approver{
			csrInterface:   kubeClient.CertificatesV1().CertificateSigningRequests(),
			serviceAccount: serviceAccount,
		}); err != nil {
		return err
	}

	go func() {
		if err := mgr.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Err(err).Msg("failed to run CSR approver")
		}
	}()

	return nil
}

// getServiceAccount pieces together the fully qualified name of the controller's service account
func getServiceAccount(env environment.Environment) (string, error) {
	var name, namespace string
	if ns := env.ControllerNamespace(); ns != nil {
		namespace = *ns
	} else {
		return "", errors.New("namespace unknown")
	}

	if sa := env.ControllerServiceAccountName(); sa != nil {
		name = *sa
	} else {
		return "", errors.New("name unknown")
	}

	return "system:serviceaccount:" + namespace + ":" + name, nil
}
