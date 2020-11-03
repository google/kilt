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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/kilt/pkg/patchset"
	"github.com/google/kilt/pkg/repo"
)

// Graph provides an interface for abstracting over a dependency graph implementation
type Graph interface {
	Add(patchset, dependency *patchset.Patchset) error
	Remove(patchset, dependency *patchset.Patchset) error
	Validate() error
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
	patchsets           repo.PatchsetCache
	reverseDependencies map[string][]*patchset.Patchset
	dependencies        map[string]*dependency
}

// NewStruct creates a new StructGraph
func NewStruct(patchsets repo.PatchsetCache) *StructGraph {
	return &StructGraph{
		patchsets:    patchsets,
		dependencies: make(map[string]*dependency),
	}
}

// Add adds a dependency to a patchset
func (d *StructGraph) Add(ps, dep *patchset.Patchset) error {
	if ps.SameAs(dep) {
		return fmt.Errorf("can't add %q as a dependency of itself", ps.Name())
	}
	if !d.checkOrder(ps, dep) {
		return fmt.Errorf("can't add %q as a dependency of preceding patchset %q", dep.Name(), ps.Name())
	}
	pdep := &patchsetPredicate{dep}
	deps, ok := d.dependencies[ps.UUID().String()]
	if !ok {
		deps = &dependency{
			patchset:   ps,
			predicates: nil,
		}
	}
	for _, p := range deps.predicates {
		if p.Patchset.SameAs(dep) {
			return fmt.Errorf("%q already exists as a dependency of %q", dep.Name(), ps.Name())
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

// flatten a structgraph to a map of patchset names to dependency names, for easy marshalling.
func (d *StructGraph) flatten() map[string][]string {
	f := map[string][]string{}
	for _, d := range d.dependencies {
		dependencies := []string{}
		for _, p := range d.predicates {
			dependencies = append(dependencies, p.String())
		}
		f[d.patchset.Name()] = dependencies
	}
	return f
}

// load a structgraph from a map of patchset names to dependendency names.
func (d *StructGraph) load(f map[string][]string) error {
	ps := make(map[string]*patchset.Patchset)
	for _, p := range d.patchsets.Slice {
		ps[p.Name()] = p
	}
	for name, deps := range f {
		p, ok := ps[name]
		if !ok {
			return fmt.Errorf("patchset %q not found", name)
		}
		dep := dependency{patchset: p}
		predicates := []*patchsetPredicate{}
		for _, depName := range deps {
			depPatchset, ok := ps[depName]
			if !ok {
				return fmt.Errorf("patchset dependency %q not found", depName)
			}
			predicates = append(predicates, &patchsetPredicate{depPatchset})
		}
		dep.predicates = predicates
		d.dependencies[p.UUID().String()] = &dep
	}
	return nil
}

// MarshalJSON implements a simple JSON marshal of a StructGraph.
func (d StructGraph) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.flatten())
}

// UnmarshalJSON implements a simple JSON unmarshal of a StructGraph.
func (d *StructGraph) UnmarshalJSON(b []byte) error {
	f := map[string][]string{}
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	return d.load(f)
}

// checkOrder verifies that dep comes before ps in the patchset list.
func (d *StructGraph) checkOrder(ps, dep *patchset.Patchset) bool {
	return d.patchsets.Index[ps.Name()] > d.patchsets.Index[dep.Name()]
}

func (d StructGraph) checkGraph() error {
	visited := make(map[string]bool)
	for _, dep := range d.dependencies {
		if visited[dep.patchset.UUID().String()] {
			continue
		}
		if ps := d.findCycles(dep, visited, make(map[string]bool)); len(ps) > 0 {
			return fmt.Errorf("cycle in dependencies: %s", strings.Join(ps, ", "))
		}
	}
	return nil
}

func (d StructGraph) findCycles(dep *dependency, permanent, temporary map[string]bool) []string {
	uuid := dep.patchset.UUID().String()
	if permanent[uuid] {
		return nil
	}
	if temporary[uuid] {
		return []string{dep.patchset.Name()}
	}

	temporary[uuid] = true

	for _, p := range dep.predicates {
		newDep, ok := d.dependencies[p.Patchset.UUID().String()]
		if !ok {
			continue
		}
		if ps := d.findCycles(newDep, permanent, temporary); len(ps) > 0 {
			return append(ps, dep.patchset.Name())
		}
	}

	delete(temporary, uuid)
	permanent[uuid] = true
	return nil
}

// Validate checks that the dependency graph is a valid DAG.
func (d StructGraph) Validate() error {
	return d.checkGraph()
}

// TransitiveDependencies will calculate a list of transitive dependencies for the patchset.
func (d StructGraph) TransitiveDependencies(ps *patchset.Patchset) []*patchset.Patchset {
	var patchsets []*patchset.Patchset
	queue := []*patchset.Patchset{ps}
	seen := map[string]struct{}{
		ps.UUID().String(): struct{}{},
	}
	for len(queue) > 0 {
		ps := queue[0]
		var predicates []*patchsetPredicate
		if dep := d.dependencies[ps.UUID().String()]; dep != nil {
			predicates = dep.predicates
		}
		for _, p := range predicates {
			patchset := p.Patchset
			if _, ok := seen[patchset.UUID().String()]; ok {
				continue
			}
			seen[patchset.UUID().String()] = struct{}{}
			patchsets = append(patchsets, patchset)
			queue = append(queue, patchset)
		}
		if len(queue) > 1 {
			queue = queue[1 : len(queue)-1]
		} else {
			break
		}
	}
	return patchsets
}

func (d *StructGraph) calculateReverseDependencies() {
	revDeps := map[string][]*patchset.Patchset{}
	for _, ps := range d.patchsets.Slice {
		revDeps[ps.UUID().String()] = nil
		var predicates []*patchsetPredicate
		if dep := d.dependencies[ps.UUID().String()]; dep != nil {
			predicates = dep.predicates
		}
		for _, p := range predicates {
			revDep := revDeps[p.Patchset.UUID().String()]
			revDeps[p.Patchset.UUID().String()] = append(revDep, ps)
		}
	}
	d.reverseDependencies = revDeps
}

// TransitiveReverseDependencies will calculate a list of transitive reverse dependencies for the patchset.
func (d *StructGraph) TransitiveReverseDependencies(ps *patchset.Patchset) []*patchset.Patchset {
	if len(d.reverseDependencies) == 0 {
		d.calculateReverseDependencies()
	}
	var patchsets []*patchset.Patchset
	queue := []*patchset.Patchset{ps}
	seen := map[string]bool{
		ps.UUID().String(): true,
	}
	for len(queue) > 0 {
		ps := queue[0]
		for _, patchset := range d.reverseDependencies[ps.UUID().String()] {
			if seen[patchset.UUID().String()] {
				continue
			}
			seen[patchset.UUID().String()] = true
			patchsets = append(patchsets, patchset)
			queue = append(queue, patchset)
		}
		if len(queue) > 1 {
			queue = queue[1 : len(queue)-1]
		} else {
			break
		}
	}
	return patchsets
}
