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
	"github.com/google/kilt/pkg/rework"

	log "github.com/golang/glog"
	"github.com/spf13/cobra"
)

var reworkCmd = &cobra.Command{
	Use:   "rework",
	Short: "Rework the patches belonging to patchsets",
	Long: `Rework patchsets, allowing patches to be redistributed and re-ordered in the
branch. The rework command will create a working area detached form the current
kilt branch where modifications can be staged without changing the original
branch.

Kilt will examine the patchsets in the branch and determine which patches
belonging to patchsets need to be reworked, and create a queue of operations
that the user will drive. The user can also perform other rework-related
operations, such as re-ordering or merging patches.

Once the user is finished, kilt will verify that the rework is valid, and
modify the previous kilt branch to point to the result of the rework. A rework
is considered valid if the end state is identical to the initial state -- the
diff between them is empty.`,
	Args: argsRework,
	Run:  runRework,
}

var reworkFlags = struct {
	begin    bool
	finish   bool
	validate bool
	force    bool
}{}

func init() {
	rootCmd.AddCommand(reworkCmd)
	reworkCmd.Flags().BoolVar(&reworkFlags.begin, "begin", true, "begin rework")
	reworkCmd.Flags().MarkHidden("begin")
	reworkCmd.Flags().BoolVar(&reworkFlags.finish, "finish", false, "validate and finish rework")
	reworkCmd.Flags().BoolVarP(&reworkFlags.force, "force", "f", false, "when finishing, force finish rework, regardless of validation")
	reworkCmd.Flags().BoolVar(&reworkFlags.validate, "validate", false, "validate rework")
}

func argsRework(*cobra.Command, []string) error {
	return nil
}

func runRework(cmd *cobra.Command, args []string) {
	var c *rework.Command
	var err error
	switch {
	case reworkFlags.finish:
		c, err = rework.NewFinishCommand(reworkFlags.force)
	case reworkFlags.validate:
		c, err = rework.NewValidateCommand()
	case reworkFlags.begin:
		c, err = rework.NewBeginCommand()
	default:
		log.Exitf("No operation specified")
	}
	if err != nil {
		log.Exitf("Rework failed: %v", err)
	}
	if err = c.Execute(); err != nil {
		log.Exitf("Rework failed: %v", err)
	}
}