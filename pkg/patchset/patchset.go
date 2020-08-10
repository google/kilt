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

// Package patchset provides functions for the creation and updating of patchsets.
package patchset

import (
	"fmt"
	"strconv"

	"github.com/pborman/uuid"
)

// Patchset represents a patchset
type Patchset struct {
	name              string
	uuid              uuid.UUID
	version           Version
	metadata          string
	patches, floating []string
}

// Version wraps a patchset version number
type Version struct {
	v int
}

// String emits a string representation of a patchset version
func (v Version) String() string {
	return fmt.Sprintf("%d", v.v)
}

// Predecessor returns the version number prior to the current version
func (v Version) Predecessor() Version {
	return Version{v.v - 1}
}

// Successor returns the version number after the current version
func (v Version) Successor() Version {
	return Version{v.v + 1}
}

// InitialVersion returns the base version number patchsets start from
func InitialVersion() Version {
	return Version{1}
}

// ParseVersion returns a Version from a string.
func ParseVersion(v string) (Version, error) {
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return Version{}, err
	}
	return Version{int(n)}, nil
}

// Cmp compares two patchset version structs and returns:
//  -1 if v1 < v2
//   0 if v1 == v2
//   1 if v1 > v2
func (v Version) Cmp(v2 Version) int {
	switch {
	case v.v < v2.v:
		return -1
	case v.v > v2.v:
		return 1
	case v.v == v2.v:
		return 0
	}
	return 0
}

// Load returns a patchset with the given fields set.
func Load(name, uuidStr string, version Version) *Patchset {
	if name == "" || uuidStr == "" {
		return nil
	}
	return &Patchset{
		name:    name,
		uuid:    uuid.Parse(uuidStr),
		version: version,
	}
}

// New creates a new patchset.
func New(name string) *Patchset {
	return Load(name, uuid.New(), InitialVersion())
}

// Version returns the version of the patchset
func (p Patchset) Version() Version {
	return p.version
}

// UUID returns the UUID of the patchset
func (p Patchset) UUID() uuid.UUID {
	return p.uuid
}

// Name returns the name of the patchset
func (p Patchset) Name() string {
	return p.name
}

// SameAs compares two patchsets and checks if they are the same, regardless of
// version or name changes.
func (p Patchset) SameAs(p2 *Patchset) bool {
	return uuid.Equal(p.uuid, p2.uuid)
}

// SameVersion compares two patchsets and checks if they are the same patchset
// and the same version of that patchset, ignoring name changes.
func (p Patchset) SameVersion(p2 *Patchset) bool {
	return p.SameAs(p2) && p.version.Cmp(p2.version) == 0
}

// Equal checks whether two patchsets are completely equal.
func (p *Patchset) Equal(p2 *Patchset) bool {
	return p.name == p2.name && p.SameVersion(p2)
}

// MetadataCommit returns the commit id of the metadata commit of the patchset.
func (p Patchset) MetadataCommit() string {
	return p.metadata
}

// AddMetadataCommit will set the metadatacommit to the given commit id.
func (p *Patchset) AddMetadataCommit(metadata string) {
	p.metadata = metadata
}

// Patches will return a list of patches in the patchset.
func (p Patchset) Patches() []string {
	return p.patches
}

// AddPatch will add the patch to the list of patches in the patchset.
func (p *Patchset) AddPatch(patch string) {
	p.patches = append(p.patches, patch)
}

// FloatingPatches will return a list of floating patches belonging to the patchset.
func (p Patchset) FloatingPatches() []string {
	return p.floating
}

// AddFloatingPatch adds the patch to the list of floating patches belonging to the patchset.
func (p *Patchset) AddFloatingPatch(patch string) {
	p.floating = append(p.floating, patch)
}
