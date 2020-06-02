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

// Package dependency implements patchset dependency support
package dependency

import (
	"fmt"

	"github.com/google/kilt/pkg/patchset"
)

// Graph provides an interface for abstracting over a dependency graph implementation
type Graph interface {
	Add(patchset, dependency *patchset.Patchset) error
	Remove(patchset, dependency *patchset.Patchset) error
}

type patchsetPredicate struct {
	Patchset *patchset.Patchset
}

func (p patchsetPredicate) String() string {
	return fmt.Sprintf("%s", p.Patchset.Name())
}

func (p patchsetPredicate) Equal(p2 *patchsetPredicate) bool {
	return p.Patchset.Equal(p2.Patchset)
}

type dependency struct {
	patchset   *patchset.Patchset
	predicates []*patchsetPredicate
}

func (d *dependency) Equal(d2 *dependency) bool {
	if !d.patchset.Equal(d2.patchset) {
		return false
	}
	if len(d.predicates) != len(d2.predicates) {
		return false
	}
	for i := range d.predicates {
		if !d.predicates[i].Equal(d2.predicates[i]) {
			return false
		}
	}
	return true
}

// StructGraph implements a simple in-memory dependency graph
type StructGraph struct {
	dependencies map[string]*dependency
}

// NewStruct creates a new StructGraph
func NewStruct() *StructGraph {
	return &StructGraph{
		dependencies: make(map[string]*dependency),
	}
}

// Add adds a dependency to a patchset
func (d *StructGraph) Add(ps, dep *patchset.Patchset) error {
	if ps.SameAs(dep) {
		return fmt.Errorf("can't add %q as a dependency of itself", ps.Name())
	}
	pdep := &patchsetPredicate{dep}
	deps, ok := d.dependencies[ps.UUID().String()]
	if !ok {
		deps = &dependency{
			patchset:   ps,
			predicates: nil,
		}
	}
	deps.predicates = append(deps.predicates, pdep)
	d.dependencies[ps.UUID().String()] = deps
	return nil
}

// Remove removes a dependency from a patchset
func (d *StructGraph) Remove(ps, dep *patchset.Patchset) error {
	ds, ok := d.dependencies[ps.UUID().String()]
	if !ok {
		return fmt.Errorf("patchset %q does not have any dependencies", ps.Name())
	}
	for i, p := range ds.predicates {
		if p.Patchset.SameAs(dep) {
			ds.predicates = append(ds.predicates[:i], ds.predicates[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("patchset %q does not depend on patchset %q", ps.Name(), dep.Name())
}
