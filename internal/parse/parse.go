// Copyright Project Contour Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parse

import (
	// Import the hash implementations or this package will panic if
	// Contour or Envoy images reference a sha hash. See the following
	// for details: https://github.com/opencontainers/go-digest#usage
	_ "crypto/sha256"
	_ "crypto/sha512"
	"fmt"
	"os/exec"
	"strings"

	"github.com/docker/distribution/reference"
)

// Image parses s, returning and error if s is not a syntactically
// valid image reference. Image does not not handle short digests.
func Image(s string) error {
	_, err := reference.Parse(s)
	if err != nil {
		return fmt.Errorf("failed to parse s %s: %w", s, err)
	}

	return nil
}

// DeploymentLogsForString parses the container logs of the specified ns/name
// deployment, returning true if the string was found.
func DeploymentLogsForString(ns, name, container, expectedString string) (bool, error) {
	cmdPath, err := exec.LookPath("kubectl")
	if err != nil {
		return false, err
	}
	slashName := fmt.Sprintf("deployment/%s", name)
	nsFlag := fmt.Sprintf("--namespace=%s", ns)
	args := []string{"logs", slashName, "-c", container, nsFlag}
	found, err := lookForString(cmdPath, args, expectedString)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}
	return false, nil
}

// lookForString looks for the given string using cmd and args, returning
// true if the string was found.
func lookForString(cmd string, args []string, expectedString string) (bool, error) {
	result, err := runCmd(cmd, args)
	if err != nil {
		return false, err
	}
	if strings.Contains(result, expectedString) {
		return true, nil
	}
	return false, nil
}

// runCmd runs command cmd with arguments args and returns the output
// of the command or an error.
func runCmd(cmd string, args []string) (string, error) {
	for _, arg := range args {
		if strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, ".") {
			return "", fmt.Errorf("invalid argument %q", arg)
		}
	}
	execCmd := exec.Command(cmd, args...)
	result, err := execCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run command %q with args %q: %v", cmd, args, err)
	}
	return string(result), nil
}
