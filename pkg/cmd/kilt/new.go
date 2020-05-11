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
	"log"
	"os"

	"github.com/google/kilt/pkg/patchset"
	"github.com/google/kilt/pkg/repo"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <patchset>",
	Short: "Create a new patchset",
	Long: `Create a new patchset in the current repo. Pass in the patchset name as the
first positional argument.`,
	Args: argsNew,
	Run:  runNew,
}

func init() {
	rootCmd.AddCommand(newCmd)
}

func argsNew(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("Patchset name required")
	}
	return nil
}

func runNew(cmd *cobra.Command, args []string) {
	log.Println("Creating new patchset")
	repo, err := repo.Open()
	if err != nil {
		log.Printf("Init failed: %s", err)
		os.Exit(-1)
	}
	ps := patchset.New(args[0])
	err = repo.AddPatchset(ps)
	if err != nil {
		log.Printf("Failed to add patchset: %s", err)
		os.Exit(-1)
	}
}
