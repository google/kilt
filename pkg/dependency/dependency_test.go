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
	d := patchset.New("d")
	e := patchset.New("e")
	patchsets := []*patchset.Patchset{c, b, a}
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
		{
			desc:     "Add reverse-dependency",
			patchset: b,
			addDeps:  []*patchset.Patchset{c, a},
			deps: map[string]*dependency{
				b.UUID().String(): {
					patchset: b,
					predicates: []*patchsetPredicate{
						{c},
					},
				},
			},
		},
		{
			desc:     "Add to non-existent patchset",
			patchset: d,
			addDeps:  []*patchset.Patchset{e},
			deps:     map[string]*dependency{},
		},
	}
	for _, tt := range tests {
		s := NewStruct(patchsets)
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
		s := NewStruct(nil)
		s.dependencies = tt.startDeps
		for _, dep := range tt.rmDeps {
			s.Remove(tt.patchset, dep)
		}
		if diff := cmp.Diff(s.dependencies, tt.endDeps); diff != "" {
			t.Errorf("%v: Remove(%v) returned diff (-got +want)\n%s", tt.desc, tt.rmDeps, diff)
		}
	}
}

func TestValidate(t *testing.T) {
	a := patchset.New("a")
	b := patchset.New("b")
	c := patchset.New("c")
	patchsets := []*patchset.Patchset{c, b, a}
	tests := []struct {
		json  []byte
		valid bool
	}{
		{[]byte(`{"a":["b","c"]}`), true},
		{[]byte(`{"a":["b","c"],"b":["a","c"]}`), false},
	}
	for _, tt := range tests {
		s := NewStruct(patchsets)
		if err := s.UnmarshalJSON(tt.json); err != nil {
			t.Errorf("Got error unmarshalling valid dependencies %q: %v", tt.json, err)
			continue
		}
		if err := s.Validate(); (err != nil) == tt.valid {
			if tt.valid {
				t.Errorf("Got error validating dependencies %q: %v", tt.json, err)
			} else {
				t.Errorf("Expected error validating dependencies %q, got nil", tt.json)
			}
		}
	}
}
