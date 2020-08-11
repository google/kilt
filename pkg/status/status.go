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

// Package status implements functions for displaying the current repo status.
package status

import (
	"fmt"

	"github.com/google/kilt/pkg/repo"
	"github.com/google/kilt/pkg/rework"
)

// Print will print the current kilt branch and rework status.
func Print() error {
	r, err := repo.Open()
	if err != nil {
		return err
	}
	fmt.Printf("On kilt branch %s with base commit %s\n", r.KiltBranch(), r.KiltBase())
	if ok, err := r.ReworkInProgress(); err != nil {
		return err
	} else if ok {
		fmt.Println("Rework in progress.")
		rework.Status(r)
		return nil
	}
	patchsets, err := r.Patchsets()
	if err != nil {
		return err
	}
	found := false
	for _, patchset := range patchsets {
		if patchset.Name() == "unknown" {
			continue
		}
		if patchset.MetadataCommit() == "" {
			fmt.Printf("Patchset %q missing metadata commit.\n", patchset.Name())
			if len(patchset.Patches()) > 0 {
				desc, err := r.DescribeCommit(patchset.Patches()[0])
				if err != nil {
					return err
				}
				fmt.Printf("First commit: %s\n", desc)
			}
		}
		if floating := patchset.FloatingPatches(); len(floating) > 0 {
			found = true
			fmt.Printf("Patchset %q needs rework; floating patches found:\n", patchset.Name())
			for i := range floating {
				desc, err := r.DescribeCommit(floating[len(floating)-i-1])
				if err != nil {
					return err
				}
				fmt.Printf("\t%s\n", desc)
			}
		}
	}
	if found {
		fmt.Println(`Rework patchsets individually using kilt rework -p <patchset>, or rework all
patches using kilt rework`)
	}
	ps, err := r.PatchsetMap()
	if err != nil {
		return err
	}
	if patchset, ok := ps["unknown"]; ok {
		fmt.Println("Patches found belonging to unknown patchset:")
		floating := patchset.FloatingPatches()
		for i := range floating {
			desc, err := r.DescribeCommit(floating[len(floating)-i-1])
			if err != nil {
				return err
			}
			fmt.Printf("\t%s\n", desc)
		}
		fmt.Println(`Please assign these patches to a patchset by adding a "Patchset-Name:" footer.`)
	}
	return nil
}
