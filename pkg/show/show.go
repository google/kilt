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

// Package show implements functions to print information about a patchset.
package show

import (
	"fmt"

	"github.com/google/kilt/pkg/repo"
)

// Patchset will print metadata and list patches for the given patchset.
func Patchset(name string) error {
	r, err := repo.Open()
	if err != nil {
		return err
	}
	patchsets, err := r.PatchsetMap()
	if err != nil {
		return err
	}
	patchset, ok := patchsets[name]
	if !ok {
		return fmt.Errorf("patchset %s not found", name)
	}
	fmt.Printf("Patchset %s, Version %s, UUID %s\n", patchset.Name(), patchset.Version(), patchset.UUID())
	fmt.Printf("Metadata commit id %s\n", patchset.MetadataCommit())
	patches := patchset.Patches()
	floating := patchset.FloatingPatches()
	if len(patches) > 0 {
		fmt.Println("Patches in patchset:")
		for _, patch := range patches {
			desc, err := r.DescribeCommit(patch)
			if err != nil {
				return err
			}
			fmt.Printf("\t%s\n", desc)
		}
	}
	if len(floating) > 0 {
		fmt.Println("Floating patches:")
		for _, patch := range floating {
			desc, err := r.DescribeCommit(patch)
			if err != nil {
				return err
			}
			fmt.Printf("\t%s\n", desc)
		}
	}
	return nil
}
