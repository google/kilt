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
	log "github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/google/kilt/pkg/status"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print the current status of the kilt branch.",
	Long: `Print the current state of the kilt branch, including printing information about
any rework in progress.

If a rework is in progress, status will display the remaining work that is
queued for the rework, as well as displaying the next suggested actions
that the user could take.

If a rework is not in progress, status will display any suggested fixes
that the user should make to the kilt branch, including reworking floating
patches or assigning unknown patches to a patchset.`,
	Args: argsStatus,
	Run:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func argsStatus(cmd *cobra.Command, args []string) error {
	return nil
}

func runStatus(cmd *cobra.Command, args []string) {
	if err := status.Print(); err != nil {
		log.Exitf("Error: %v", err)
	}
}
