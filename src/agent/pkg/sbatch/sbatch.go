// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package sbatch

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"regexp"

	"al.essio.dev/pkg/shellescape"
	"github.com/supernetes/supernetes/agent/pkg/agent"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/config/pkg/config"
)

/*
srun --account project_123456789 --partition standard -- sh -c 'echo "Hello from $(hostname)!"'
srun --account project_123456789 --partition standard -- singularity exec --compat docker://alpine:latest sh -c 'for i in $(seq 10); do echo $i; sleep 1; done'
scontrol show job --json | jq ".jobs[].job_resources.nodes"
*/

type Runtime interface {
	// Run dispatches the given workload, returning its tracking ID
	Run(workload *api.Workload) (string, error)
}

type runtime struct {
	config           *config.SlurmConfig
	containerRuntime string
}

var _ Runtime = &runtime{} // Static type assert

func NewRuntime(config *config.SlurmConfig) Runtime {
	return &runtime{
		config:           config,
		containerRuntime: containerRuntime(),
	}
}

var jobIdRegex = regexp.MustCompile(`^Submitted batch job (\\d+)$`)

func (r *runtime) Run(workload *api.Workload) (string, error) {
	output := bytes.NewBuffer(nil)
	cmd := exec.Command("sbatch") // `srun` can only run synchronously
	cmd.Stdin = bytes.NewReader([]byte(r.composeScript(workload)))
	cmd.Stdout = output
	if err := cmd.Run(); err != nil {
		return "", err
	}

	match := jobIdRegex.FindSubmatch(output.Bytes())
	if len(match) != 2 {
		return "", fmt.Errorf("failed to parse scontrol output")
	}

	return string(match[1]), nil
}

func (r *runtime) composeScript(workload *api.Workload) string {
	sbatchOpts := map[string]string{
		"account":   r.config.Account,
		"partition": r.config.Partition,
		// https://slurm.schedmd.com/sbatch.html#SECTION_FILENAME-PATTERN
		"output": path.Join(agent.IoDir(), "%j.stout"),
		"error":  path.Join(agent.IoDir(), "%j.stderr"),
	}

	script := "#!/bin/sh\n"
	for k, v := range sbatchOpts {
		script += fmt.Sprintf("#SBATCH --%s %q\n", k, v)
	}

	script += shellescape.QuoteCommand(append(
		[]string{r.containerRuntime, "exec", "--compat", fmt.Sprintf("docker://%s", workload.Spec.Image)},
		workload.Spec.Args...,
	))

	log.Debug().Str("script", script).Msg("composed sbatch script")
	return script
}
