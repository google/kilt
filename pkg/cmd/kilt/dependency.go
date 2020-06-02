/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kilt

import (
	"errors"

	log "github.com/golang/glog"

	"github.com/google/kilt/pkg/dependency"
	"github.com/google/kilt/pkg/patchset"
	"github.com/google/kilt/pkg/repo"

	"github.com/spf13/cobra"
)

var addDepCmd = &cobra.Command{
	Use:   "add-dep <patchset> <p1> [p2...]",
	Short: "Add a dependency to a patchset",
	Long: `Add one or more dependencies to a patchset. Pass in multiple patchset names to
include multiple dependencies.`,
	Args: argsDep,
	Run:  runAdd,
}

var rmDepCmd = &cobra.Command{
	Use:   "rm-dep <patchset> <p1> [p2...]",
	Short: "Remove a dependency from a patchset",
	Long: `Remove one or more dependencies to a patchset. Pass in multiple patchset names to
include multiple dependencies.`,
	Args: argsDep,
	Run:  runRm,
}

func init() {
	rootCmd.AddCommand(addDepCmd)
	rootCmd.AddCommand(rmDepCmd)
}

func argsDep(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return errors.New("Patchset name and at least one dependency required")
	}
	return nil
}

func runAdd(cmd *cobra.Command, args []string) {
	runDep(dependency.Graph.Add, cmd, args)
}

func runRm(cmd *cobra.Command, args []string) {
	runDep(dependency.Graph.Remove, cmd, args)
}

func runDep(op func(d dependency.Graph, ps, dep *patchset.Patchset) error, cmd *cobra.Command, args []string) {
	deps := dependency.NewStruct()
	repo, err := repo.Open()
	if err != nil {
		log.Exitf("Init failed: %s", err)
	}
	ps, err := repo.FindPatchset(args[0])
	if err != nil {
		log.Exitf("Patchset %q does not exist: %v", args[0], err)
	}
	for _, d := range args[1:] {
		dep, err := repo.FindPatchset(d)
		if err != nil {
			log.Exitf("Dependency %q does not exist: %v", args[0], err)
		}
		if err = op(deps, ps, dep); err != nil {
			log.Exitf("Operation failed: %v", err)
		}
	}
}
