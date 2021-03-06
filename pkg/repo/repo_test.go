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

package repo

import (
	"os"
	"testing"

	"github.com/google/kilt/pkg/internal/testfiles"
	"github.com/google/kilt/pkg/patchset"

	"github.com/libgit2/git2go/v30"
)

func setupRepo(t *testing.T, name string) *git.Repository {
	path, err := testfiles.TempDir(name)
	if err != nil {
		t.Fatalf("TempDir(): %v", err)
	}
	os.Chdir(path)
	repo, err := git.InitRepository(path, false)
	if err != nil {
		t.Fatalf("InitRepository(): %v", err)
	}
	config, err := repo.Config()
	if err != nil {
		t.Fatalf("Config(): %v", err)
	}
	if err = config.SetString("user.name", "Test Data"); err != nil {
		t.Fatalf("SetString(): %v", err)
	}
	if err = config.SetString("user.email", "nobody@google.com"); err != nil {
		t.Fatalf("SetString(): %v", err)
	}
	index, err := repo.Index()
	if err != nil {
		t.Fatalf("Index(): %v", err)
	}
	oid, err := index.WriteTree()
	if err != nil {
		t.Fatalf("WriteTree(): %v", err)
	}
	tree, err := repo.LookupTree(oid)
	if err != nil {
		t.Fatalf("LookupTree(): %v", err)
	}
	sig, err := repo.DefaultSignature()
	if err != nil {
		t.Fatalf("DefaultSignature(): %v", err)
	}
	oid, err = repo.CreateCommit("HEAD", sig, sig, "Initial commit.", tree)
	if err != nil {
		t.Fatalf("CreateCommit(): %v", err)
	}
	commit, err := repo.LookupCommit(oid)
	if err != nil {
		t.Fatalf("LookupCommit(): %v", err)
	}
	branch, err := repo.CreateBranch("test", commit, false)
	if err != nil {
		t.Fatalf("CreateBranch(): %v", err)
	}
	err = repo.SetHead(branch.Reference.Name())
	if err != nil {
		t.Fatalf("SetHead(): %v", err)
	}

	return repo
}

func cleanupRepo(t *testing.T, repo *git.Repository) {
	tmp := repo.Workdir()
	err := os.RemoveAll(tmp)
	if err != nil {
		t.Fatalf("RemoveAll(): %v", err)
	}
}

func TestCreateMetadataCommit(t *testing.T) {
	r := setupRepo(t, "CreateMetadataCommit")
	defer cleanupRepo(t, r)
	g := newWithGitRepo(r, "", "test", "test")
	ref, err := g.git.Head()
	if err != nil {
		t.Fatalf("Head(): %v", err)
	}
	ps := patchset.New("test")
	err = g.createMetadataCommit(ps)
	if err != nil {
		t.Fatalf("createMetadataCommit(): %v", err)
	}
	newref, err := g.git.Head()
	if err != nil {
		t.Fatalf("Head(): %v", err)
	}
	if ref.Cmp(newref) == 0 {
		t.Fatalf("createMetadataCommit(): No metadata created")
	}
}

func TestPatchsetMap(t *testing.T) {
	r := setupRepo(t, "CreateMetadataCommit")

	head, err := r.Head()
	if err != nil {
		t.Fatalf("r.Head(): %v", err)
	}

	headCommit, err := head.Peel(git.ObjectCommit)
	if err != nil {
		t.Fatalf("head.Peel(): %v", err)
	}

	base := headCommit.Id().String()
	defer cleanupRepo(t, r)

	g := newWithGitRepo(r, base, "test", "test")

	patchsets := []string{"a", "b", "c"}
	tests := []struct {
		in  string
		out bool
	}{
		{"a", true},
		{"b", true},
		{"c", true},
		{"d", false},
		{"", false},
	}
	for _, p := range patchsets {
		ps := patchset.New(p)
		if err := g.createMetadataCommit(ps); err != nil {
			t.Fatalf("createMetadataCommit(%q): %v", p, err)
		}
	}
	ps, err := g.PatchsetMap()
	if err != nil {
		t.Fatalf("PatchsetMap(): %v", err)
	}
	for _, tt := range tests {
		p := ps[tt.in]
		switch {
		case p == nil && tt.out:
			t.Errorf("PatchsetMap[%q]: Got unexpected nil", tt.in)
		case p != nil && !tt.out:
			t.Errorf("PatchsetMap[%q]: Got patchset, expected nil", tt.in)
		}
	}
}
