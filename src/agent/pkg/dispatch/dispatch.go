// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package dispatch

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/rs/zerolog"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	sulog "github.com/supernetes/supernetes/common/pkg/log"
	"google.golang.org/protobuf/proto"
)

func Run(containerSpecs string) {
	containers, err := decodeContainerSpecs(containerSpecs)
	sulog.FatalErr(err).Msg("decoding container specifications failed")

	runtime := containerRuntime()
	output := &outputBuffer{}
	exitCodes := make(chan int)

	for _, container := range containers {
		log := sulog.Scoped().Str("container", container.Name).Logger()
		dispatchContainer(output, runtime, container, exitCodes, &log)
	}

	var code int
	for range containers {
		code = max(code, <-exitCodes) // Highest exit code wins
	}

	os.Exit(code)
}

type outputBuffer struct {
	mutex sync.Mutex
}

func (o *outputBuffer) writeLine(line string) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	_, _ = os.Stdout.Write([]byte(line))
}

func (o *outputBuffer) readFromStream(stream io.Reader, formatter func(string) string) {
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		o.writeLine(formatter(scanner.Text()))
	}
}

func dispatchContainer(output *outputBuffer, runtime string, container *api.WorkloadContainer, exitCodes chan<- int, log *zerolog.Logger) {
	command := "run"
	if len(container.Command) > 0 {
		command = "exec" // Allows for overriding the container ENTRYPOINT
	}

	args := append(
		[]string{command, "--compat", fmt.Sprintf("docker://%s", container.Image)},
		append(container.Command, container.Args...)...,
	)

	log.Debug().Str("command", runtime).Strs("args", args).Msg("composed command")
	cmd := exec.Command(runtime, args...)

	var err error
	var stdoutPipe, stderrPipe io.Reader

	if stdoutPipe, err = cmd.StdoutPipe(); err != nil {
		log.Fatal().Err(err).Msg("initializing stdout pipe failed")
	}

	if stderrPipe, err = cmd.StderrPipe(); err != nil {
		log.Fatal().Err(err).Msg("initializing stderr pipe failed")
	}

	for _, pipe := range []io.Reader{stdoutPipe, stderrPipe} {
		go output.readFromStream(pipe, func(line string) string {
			return fmt.Sprintf("%s %s %s\n", time.Now().Format(time.RFC3339), container.Name, line)
		})
	}

	go func() {
		var exitCode int
		if err := cmd.Run(); err != nil {
			log.Debug().Err(err).Msg("command failed")

			exitCode = 1 // Generic failure
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				// More specific exit code
				exitCode = exitError.ExitCode()
			}
		}

		// Submit the exit code
		exitCodes <- exitCode
	}()
}

func decodeContainerSpecs(containerSpecs string) ([]*api.WorkloadContainer, error) {
	b, err := base64.StdEncoding.DecodeString(containerSpecs)
	if err != nil {
		return nil, err
	}

	containers := &api.WorkloadContainers{}
	if err := proto.Unmarshal(b, containers); err != nil {
		return nil, err
	}

	return containers.Array, nil
}
