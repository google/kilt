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

package dependency

import (
	"testing"

	"github.com/google/kilt/pkg/patchset"

	"github.com/google/go-cmp/cmp"
)

func TestAdd(t *testing.T) {
	a := patchset.New("a")
	b := patchset.New("b")
	c := patchset.New("c")
	tests := []struct {
		desc     string
		patchset *patchset.Patchset
		addDeps  []*patchset.Patchset
		deps     map[string]*dependency
	}{
		{
			desc:     "Add normal dependencies",
			patchset: a,
			addDeps:  []*patchset.Patchset{b, c},
			deps: map[string]*dependency{
				a.UUID().String(): {
					patchset: a,
					predicates: []*patchsetPredicate{
						{b},
						{c},
					},
				},
			},
		},
		{
			desc:     "Add self-dependency",
			patchset: a,
			addDeps:  []*patchset.Patchset{a, c},
			deps: map[string]*dependency{
				a.UUID().String(): {
					patchset: a,
					predicates: []*patchsetPredicate{
						{c},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		s := NewStruct()
		for _, dep := range tt.addDeps {
			s.Add(tt.patchset, dep)
		}
		if diff := cmp.Diff(s.dependencies, tt.deps); diff != "" {
			t.Errorf("%v: Add(%v) returned diff (-got +want)\n%s", tt.desc, tt.addDeps, diff)
		}
	}
}

func TestRemove(t *testing.T) {
	a := patchset.New("a")
	b := patchset.New("b")
	c := patchset.New("c")
	tests := []struct {
		desc      string
		patchset  *patchset.Patchset
		rmDeps    []*patchset.Patchset
		startDeps map[string]*dependency
		endDeps   map[string]*dependency
	}{
		{
			desc:     "Remove existing dependency",
			patchset: a,
			rmDeps:   []*patchset.Patchset{b},
			startDeps: map[string]*dependency{
				a.UUID().String(): {
					patchset: a,
					predicates: []*patchsetPredicate{
						{b},
						{c},
					},
				},
			},
			endDeps: map[string]*dependency{
				a.UUID().String(): {
					patchset: a,
					predicates: []*patchsetPredicate{
						{c},
					},
				},
			},
		},
		{
			desc:     "Remove non-existant dependency",
			patchset: a,
			rmDeps:   []*patchset.Patchset{a},
			startDeps: map[string]*dependency{
				a.UUID().String(): {
					patchset: a,
					predicates: []*patchsetPredicate{
						{b},
						{c},
					},
				},
			},
			endDeps: map[string]*dependency{
				a.UUID().String(): {
					patchset: a,
					predicates: []*patchsetPredicate{
						{b},
						{c},
					},
				},
			},
		},
		{
			desc:     "Remove non-existant dependency from empty list",
			patchset: a,
			rmDeps:   []*patchset.Patchset{b},
			startDeps: map[string]*dependency{
				a.UUID().String(): {
					patchset:   a,
					predicates: []*patchsetPredicate{},
				},
			},
			endDeps: map[string]*dependency{
				a.UUID().String(): {
					patchset:   a,
					predicates: []*patchsetPredicate{},
				},
			},
		},
	}
	for _, tt := range tests {
		s := NewStruct()
		s.dependencies = tt.startDeps
		for _, dep := range tt.rmDeps {
			s.Remove(tt.patchset, dep)
		}
		if diff := cmp.Diff(s.dependencies, tt.endDeps); diff != "" {
			t.Errorf("%v: Remove(%v) returned diff (-got +want)\n%s", tt.desc, tt.rmDeps, diff)
		}
	}
}