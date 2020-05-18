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
	log.Exit("add dep placeholder")
}

func runRm(cmd *cobra.Command, args []string) {
	log.Exit("rm dep placeholder")
}
