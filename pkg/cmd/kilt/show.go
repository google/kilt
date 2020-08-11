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
	"github.com/google/kilt/pkg/show"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Print information about a specific patchset",
	Long: `Display information about a patchset, showing the user patchset metadata
(version, UUID, name), the id and a short description of each component patch
of the patchset, as well as any floating patches that belong to the patchset.`,
	Args: argsShow,
	Run:  runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func argsShow(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("at least one patchset name is required")
	}
	return nil
}

func runShow(cmd *cobra.Command, args []string) {
	for _, arg := range args {
		if err := show.Patchset(arg); err != nil {
			log.Exitf("Error: %v", err)
		}
	}
}
