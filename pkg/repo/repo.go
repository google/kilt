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

// Package repo manages interfacing with a repo
package repo

import (
	"fmt"

	"github.com/libgit2/git2go/v28"
	"github.com/google/kilt/pkg/patchset"
)

// Repo wraps git repo state for repository manipulations
type Repo struct {
	git *git.Repository
}

const (
	metadataMessage = "kilt metadata: patchset %s\n\nPatchset-Name: %s\nPatchset-UUID: %s"
)

// Open tries to open a repo in the current working directory
func Open() (*Repo, error) {
	g, err := git.OpenRepository(".")
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}
	return &Repo{git: g}, nil
}

// AddPatchset will add the given patchset to the head of the repo
func (r *Repo) AddPatchset(ps *patchset.Patchset) error {
	err := r.createMetadataCommit(ps)
	return err
}

func (r *Repo) createMetadataCommit(ps *patchset.Patchset) error {
	head, err := r.git.Head()
	if err != nil {
		return fmt.Errorf("failed to get repo head: %w", err)
	}
	obj, err := head.Peel(git.ObjectCommit)
	if err != nil {
		return fmt.Errorf("failed to get head object: %w", err)
	}
	commit, err := obj.AsCommit()
	if err != nil {
		return fmt.Errorf("failed to get head commit: %w", err)
	}
	sig, err := r.git.DefaultSignature()
	if err != nil {
		return fmt.Errorf("failed to get default signature: %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get commit tree: %w", err)
	}
	message := fmt.Sprintf(metadataMessage, ps.Name, ps.Name, ps.UUID)
	_, err = r.git.CreateCommit(head.Branch().Reference.Name(), sig, sig, message, tree, commit)
	if err != nil {
		return fmt.Errorf("failed to create new commit: %w", err)
	}
	return nil
}
