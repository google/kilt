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

package patchset

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNew(t *testing.T) {
	tests := []struct {
		in  string
		out *Patchset
	}{
		{"patchset", &Patchset{Name: "patchset"}},
		{"", nil},
	}
	for _, tt := range tests {
		ps := New(tt.in)
		if ps != nil {
			if ps.UUID == nil {
				t.Errorf("New(%v) returned nil UUID", ps)
			}
			ps.UUID = nil
		}
		if diff := cmp.Diff(ps, tt.out); diff != "" {
			t.Errorf("New(%v) returned diff (-want +got):\n%s", ps, diff)
		}
	}
}
