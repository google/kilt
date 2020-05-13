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

// Package kilt initialize subcommands for kilt.
package kilt

import (
	log "github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/google/kilt/pkg/cmd/kilt/internal/flag"
)

var rootCmd = &cobra.Command{
	Use:   "kilt",
	Short: "kilt is a patchset management tool",
	Long:  "kilt is a tool for managing patches and patchsets.",
}

// Execute is the entry point into subcommand processing.
func Execute() {
	flag.AddFlags()
	if err := rootCmd.Execute(); err != nil {
		log.Exitf("Error: %s", err)
	}
}
