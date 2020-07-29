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

	"github.com/google/kilt/pkg/repo"

	log "github.com/golang/glog"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <base>",
	Short: "Initialize branch to work with Kilt",
	Long: `Initialize the current branch to work with Kilt. Pass in a <base> specified in
the form of a git revision. Every commit on top of <base> can be managed by Kilt. `,
	Args: argsInit,
	Run:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func argsInit(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("<base> required")
	}
	return nil
}

func runInit(cmd *cobra.Command, args []string) {
	_, err := repo.Init(args[0])
	if err != nil {
		log.Errorf("Failed to initialize Kilt: %v", err)
	}
}
