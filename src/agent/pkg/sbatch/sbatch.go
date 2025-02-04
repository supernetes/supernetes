// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package sbatch

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"al.essio.dev/pkg/shellescape"
	"github.com/pkg/errors"
	"github.com/supernetes/supernetes/agent/pkg/cache"
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

var jobIdRegex = regexp.MustCompile(`^Submitted batch job (\d+)\n$`)

func (r *runtime) Run(workload *api.Workload) (string, error) {
	script, err := r.composeScript(workload)
	if err != nil {
		log.Err(err).Msg("failed to compose sbatch script")
		return "", err
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd := exec.Command("sbatch") // `srun` can only run synchronously
	cmd.Stdin = bytes.NewReader([]byte(script))
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		log.Error().
			Bytes("stderr", stderr.Bytes()).
			Msg("sbatch execution failed")

		return "", err
	}

	match := jobIdRegex.FindSubmatch(stdout.Bytes())
	if len(match) != 2 {
		log.Error().
			Bytes("stdout", stdout.Bytes()).
			Msg("sbatch didn't produce the expected output")

		return "", errors.New("failed to parse sbatch output")
	}

	return string(match[1]), nil
}

func (r *runtime) composeScript(workload *api.Workload) (string, error) {
	sbatchOpts := map[string]string{
		"job-name":  workload.Meta.Name,
		"account":   r.config.Account,
		"partition": r.config.Partition,
		// https://slurm.schedmd.com/sbatch.html#SECTION_FILENAME-PATTERN
		"output":   path.Join(cache.IoDir(), "%j.out"),
		"nodelist": strings.Join(workload.Spec.NodeNames, ","),
	}

	extraOpts := make(map[string]string)
	for option, value := range workload.Spec.JobOptions {
		// Overriding the Supernetes-managed options is not permitted
		if _, ok := sbatchOpts[option]; ok {
			return "", fmt.Errorf("overriding option %q is not permitted", option)
		}

		extraOpts[option] = value
	}

	if len(workload.Spec.NodeNames) == 0 {
		delete(sbatchOpts, "nodelist") // No node list was given
	}

	// Incorporate the extra options for sbatch
	maps.Copy(sbatchOpts, extraOpts)

	script := "#!/bin/bash\n"
	for k, v := range sbatchOpts {
		script += fmt.Sprintf("#SBATCH --%s %q\n", k, v)
	}

	// Safety options for the actual command
	script += "set -eo pipefail\n"

	command := "run"
	if len(workload.Spec.Command) > 0 {
		command = "exec" // Allows for overriding the container ENTRYPOINT
	}

	script += shellescape.QuoteCommand(append(
		[]string{r.containerRuntime, command, "--compat", fmt.Sprintf("docker://%s", workload.Spec.Image)},
		append(workload.Spec.Command, workload.Spec.Args...)...,
	))

	binPath, err := agentPath()
	if err != nil {
		return "", errors.Wrap(err, "resolving agent binary path failed")
	}

	// Append the timestamping provided by the agent
	script += fmt.Sprintf(" |& %q timestamp", binPath)

	log.Debug().Str("script", script).Msg("composed sbatch script")
	return script, nil
}

func agentPath() (string, error) {
	binPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.EvalSymlinks(binPath)
}
