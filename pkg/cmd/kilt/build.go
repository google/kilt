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

	"github.com/google/kilt/pkg/rework"

	log "github.com/golang/glog"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build a new tree using the specified patchsets.",
	Long:  `build a new tree using the specified patchsets.`,
	Args:  argsbuild,
	Run:   runbuild,
}

var buildFlags = struct {
	begin     bool
	finish    bool
	validate  bool
	rContinue bool
	abort     bool
	skip      bool
	force     bool
	auto      bool
	patchsets []string
	all       bool
	base      string
}{}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().BoolVar(&buildFlags.begin, "begin", true, "begin rework")
	buildCmd.Flags().MarkHidden("begin")
	buildCmd.Flags().BoolVar(&buildFlags.abort, "abort", false, "abort rework")
	buildCmd.Flags().BoolVar(&buildFlags.rContinue, "continue", false, "continue rework")
	buildCmd.Flags().StringSliceVarP(&buildFlags.patchsets, "patchset", "p", nil, "specify individual patchset for rework")
	buildCmd.Flags().StringVarP(&buildFlags.base, "base", "b", "", "specify base")
}

func argsbuild(cmd *cobra.Command, args []string) error {
	if buildFlags.abort || buildFlags.rContinue {
		return nil
	}
	if len(buildFlags.patchsets) == 0 {
		return errors.New("Must specify at least one patchset")
	}
	if buildFlags.base == "" {
		return errors.New("Must specify valid base")
	}
	return nil
}

func runbuild(cmd *cobra.Command, args []string) {
	var c *rework.Command
	var err error
	switch {
	case buildFlags.finish:
		buildFlags.auto = true
		c, err = rework.NewFinishCommand(buildFlags.force)
	case buildFlags.abort:
		c, err = rework.NewAbortCommand()
	case buildFlags.skip:
		c, err = rework.NewSkipCommand()
	case buildFlags.rContinue:
		c, err = rework.NewContinueCommand()
	case buildFlags.begin:
		var targets []rework.TargetSelector
		for _, p := range buildFlags.patchsets {
			targets = append(targets, rework.PatchsetTarget{Name: p})
		}
		c, err = rework.NewBeginBuildCommand(buildFlags.base, targets...)
	default:
		log.Exitf("No operation specified")
	}
	if err != nil {
		log.Exitf("Rework failed: %v", err)
	}
	err = c.ExecuteAll()
	if err != nil {
		log.Exitf("Rework failed: %v", err)
	}
	if err := c.Save(); err != nil {
		log.Exitf("Failed to save rework state: %v", err)
	}
}
