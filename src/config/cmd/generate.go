// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"time"

	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/config/pkg/run"
)

func NewCmdGenerate() *cobra.Command {
	flags := run.NewGenerateFlags()

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate linked configuration files for a controller and an agent",
		Long: dedent.Dedent(`
			Generate linked configuration files for a controller and an agent. The
			configuration file paths are specified by the --controller-config and
			--agent-config flags respectively.

			WARNING: Existing controller and agent configuration files will be overwritten!

			Example usage:
			    $ config generate --slurm-account project_123456789 --slurm-partition standard
			    $ config generate ... # Results in controller.yaml (K8s Secret) and agent.yaml
			    $ config generate --secret=false ... # controller.yaml as plain YAML
			    $ config generate \
			        --agent-config my-agent.yaml \
			        --agent-endpoint supernetes.example.com:443 \
			        --controller-config my-controller-secret.yaml \
			        --controller-port 12345 \
			        --controller-secret-name custom-supernetes-config \
			        --controller-secret-namespace custom-supernetes-namespace \
					--reconcile-nodes 10s \
					--reconcile-workloads 10s \
			        --slurm-account project_123456789 \
			        --slurm-partition standard \
			        --filter-partition '^(?:standard)|(?:bench)$' \
			        --filter-node '^nid0010[0-9]{2}$' \
			        --cert-days-valid 365
		`),
		Run: func(cmd *cobra.Command, args []string) {
			options, err := flags.NewGenerateOptions(args, cmd.Flags())
			log.FatalErr(err).Msg("failed to parse options")
			log.FatalErr(run.Generate(options)).Msg("failed to run generate")
		},
	}

	addGenerateFlags(cmd.Flags(), flags)
	return cmd
}

func addGenerateFlags(fs *pflag.FlagSet, flags *run.GenerateFlags) {
	fs.StringVarP(&flags.AgentConfigPath, "agent-config", "a", "agent.yaml", "agent configuration file path")
	fs.StringVarP(&flags.AgentEndpoint, "agent-endpoint", "e", "localhost:40404", "endpoint agent should connect to, as specified by https://github.com/grpc/grpc/blob/18c42a21af2331c4c755257a968490ab74c587b7/doc/naming.md")

	fs.StringVarP(&flags.ControllerConfigPath, "controller-config", "c", "controller.yaml", "controller configuration file path")
	fs.Uint16VarP(&flags.ControllerPort, "controller-port", "p", 40404, "listening port for the controller")
	fs.BoolVarP(&flags.ControllerSecret, "controller-secret", "s", true, "output controller configuration as a Kubernetes Secret")
	fs.StringVar(&flags.ControllerSecretName, "controller-secret-name", "supernetes-config", "name of the controller configuration Secret")
	fs.StringVar(&flags.ControllerSecretNamespace, "controller-secret-namespace", "supernetes", "namespace of the controller configuration Secret")

	fs.DurationVar(&flags.NodeReconciliationInterval, "reconcile-nodes", time.Minute, "node reconciliation interval")
	fs.DurationVar(&flags.WorkloadReconciliationInterval, "reconcile-workloads", time.Minute, "workload reconciliation interval")

	fs.StringVar(&flags.SlurmAccount, "slurm-account", "", "default Slurm partition to use for dispatching jobs")
	fs.StringVar(&flags.SlurmPartition, "slurm-partition", "", "default Slurm partition to use for dispatching jobs")

	fs.StringVar(&flags.FilterPartitionRegex, "filter-partition", "", "Regex that limits the agent to consider only specific partitions")
	fs.StringVar(&flags.FilterNodeRegex, "filter-node", "", "Regex that limits the agent to consider only specific nodes")

	fs.Uint32VarP(&flags.CertDaysValid, "cert-days-valid", "d", 3650, "validity period of the mTLS certificates in days")
}
