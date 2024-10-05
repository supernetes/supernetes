// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/yaml"
)

// MTlsConfig stores PEM-encoded certificates and keys for one party in mTLS
type MTlsConfig struct {
	Ca   string `json:"ca"`   // CA certificate used to validate the other party
	Key  string `json:"key"`  // Private key of this party
	Cert string `json:"cert"` // Certificate of this party
}

// Decode a configuration struct from the given YAML bytes
func Decode[T any](input []byte) (*T, error) {
	var config T
	if err := yaml.UnmarshalStrict(input, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// Encode a configuration struct into YAML bytes
func Encode(config any) ([]byte, error) {
	return yaml.Marshal(config)
}

// EncodeK8s encodes a corev1 runtime.Object into YAML bytes
func EncodeK8s(obj runtime.Object) ([]byte, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	yamlSerializer := json.NewSerializerWithOptions(
		json.DefaultMetaFactory,
		scheme,
		scheme,
		json.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)

	factory := serializer.NewCodecFactory(scheme)
	encoder := factory.EncoderForVersion(yamlSerializer, corev1.SchemeGroupVersion)

	return runtime.Encode(encoder, obj)
}
