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
	"regexp"
	"strings"

	log "github.com/golang/glog"

	"github.com/libgit2/git2go/v28"
	"github.com/google/kilt/pkg/patchset"
)

// Repo wraps git repo state for repository manipulations
type Repo struct {
	git         *git.Repository
	base        string
	patchsets   []*patchset.Patchset
	patchsetMap map[string]*patchset.Patchset
}

const (
	metadataPrefix  = "kilt metadata: patchset "
	metadataMessage = metadataPrefix + `%s

Patchset-Name: %s
Patchset-UUID: %s
Patchset-Version: %s
`
)

var (
	fieldsRegexp = regexp.MustCompile("^([-[:alnum:]]+):[[:space:]]?(.*)$")
)

func newWithGitRepo(git *git.Repository, base string) *Repo {
	return &Repo{
		git:  git,
		base: base,
	}
}

// Open tries to open a repo in the current working directory
func Open() (*Repo, error) {
	g, err := git.OpenRepository(".")
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}
	base, err := g.References.Lookup("refs/kilt/base")
	if err != nil {
		return nil, fmt.Errorf("failed to lookup base: %w", err)
	}
	return newWithGitRepo(g, base.Target().String()), nil
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
	message := fmt.Sprintf(metadataMessage, ps.Name(), ps.Name(), ps.UUID(), ps.Version())
	_, err = r.git.CreateCommit(head.Branch().Reference.Name(), sig, sig, message, tree, commit)
	if err != nil {
		return fmt.Errorf("failed to create new commit: %w", err)
	}
	return nil
}

// FindPatchset iterates through the git tree and attempts to find the named patchset.
func (r *Repo) FindPatchset(name string) (*patchset.Patchset, error) {
	if len(r.patchsets) == 0 {
		if err := r.walkPatchsets(); err != nil {
			return nil, err
		}
	}
	for _, p := range r.patchsets {
		if p.Name() == name {
			return p, nil
		}
	}
	return nil, nil
}

func (r *Repo) walkPatchsets() error {
	head, err := r.git.Head()
	if err != nil {
		return err
	}
	headCommit, err := head.Peel(git.ObjectCommit)
	if err != nil {
		return err
	}
	revWalk, err := r.git.Walk()
	if err != nil {
		return err
	}
	defer revWalk.Free()

	revWalk.Sorting(git.SortTopological | git.SortTime)

	if err := revWalk.Push(headCommit.Id()); err != nil {
		return err
	}

	baseObj, err := r.git.RevparseSingle(r.base)
	if err != nil {
		return err
	}

	if err := revWalk.Hide(baseObj.Id()); err != nil {
		return err
	}

	var oid git.Oid
	var patchsets []*patchset.Patchset
	patchsetMap := map[string]*patchset.Patchset{}
	for {
		if err := revWalk.Next(&oid); err != nil {
			break
		}

		c, err := r.git.LookupCommit(&oid)
		if err != nil {
			return err
		}

		if c.ParentCount() != 1 {
			continue
		}

		if isMetadataCommit(c) {
			fields := parseFields(c.Message())
			name, ok := fields["Patchset-Name"]
			if !ok {
				log.Warningf("Error parsing metadata: no Patchset-Name field found on commit %q", c.Id())
				continue
			}
			uuid, ok := fields["Patchset-UUID"]
			if !ok {
				log.Warningf("Error parsing metadata: no Patchset-UUID field found on commit %q", c.Id())
				continue
			}
			version := patchset.InitialVersion()
			v, ok := fields["Patchset-Version"]
			if !ok {
				log.Warningf("Error parsing metadata: no Patchset-Version field found on commit %q", c.Id())
			}
			if parsedVersion, err := patchset.ParseVersion(v); err != nil {
				log.Warningf("Error parsing version for commit %q: %v", c.Id(), err)
			} else {
				version = parsedVersion
			}
			if _, ok := patchsetMap[name]; ok {
				log.Warningf("Patchset %q seen twice", name)
				continue
			}
			patchset := patchset.Load(name, uuid, version)
			patchsets = append(patchsets, patchset)
			patchsetMap[name] = patchset
		}
	}
	r.patchsets = patchsets
	r.patchsetMap = patchsetMap
	return nil
}

func isMetadataCommit(commit *git.Commit) bool {
	return strings.HasPrefix(commit.Message(), metadataPrefix)
}

func parseFields(message string) map[string]string {
	fields := map[string]string{}
	for _, l := range strings.Split(message, "\n")[1:] {
		if f := fieldsRegexp.FindStringSubmatch(l); len(f) == 3 {
			fields[f[1]] = f[2]
		}
	}
	return fields
}
