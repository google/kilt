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
	"errors"
	"fmt"
	"path"
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
	metadataPrefix       = "kilt metadata: patchset "
	patchsetNameField    = "Patchset-Name"
	patchsetUUIDField    = "Patchset-UUID"
	patchsetVersionField = "Patchset-Version"
	metadataMessage      = metadataPrefix + "%s\n\n" + patchsetNameField + ": %s\n" + patchsetUUIDField + ": %s\n" + patchsetVersionField + ": %s\n"
	refPath              = "refs/kilt"
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
	baseRefPath, err := baseRef(g)
	if err != nil {
		return nil, fmt.Errorf("failed to generate base ref: %w", err)
	}
	base, err := g.References.Lookup(baseRefPath)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup base: %w", err)
	}
	return newWithGitRepo(g, base.Target().String()), nil
}

// Init initializes kilt in the current branch.
func Init(base string) (*Repo, error) {
	g, err := git.OpenRepository(".")
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}
	obj, err := g.RevparseSingle(base)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base %q: %w", base, err)
	}
	baseRefPath, err := baseRef(g)
	if err != nil {
		return nil, fmt.Errorf("failed to generate base ref: %w", err)
	}
	if _, err := g.References.Create(baseRefPath, obj.Id(), false, fmt.Sprintf("Creating kilt base reference %s", baseRefPath)); err != nil {
		return nil, fmt.Errorf("failed to create ref: %w", err)
	}
	return newWithGitRepo(g, base), nil
}

func baseRef(g *git.Repository) (string, error) {
	if detached, err := g.IsHeadDetached(); err != nil {
		return "", fmt.Errorf("failed while checking detached head: %w", err)
	} else if detached {
		return "", errors.New("must not be on a detached head")
	}
	head, err := g.Head()
	if err != nil {
		return "", fmt.Errorf("failed to read head: %w", err)
	}
	branch := head.Branch()
	branchName, err := branch.Name()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch name: %w", err)
	}
	return path.Join(refPath, branchName, "base"), nil
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
	patchsets, err := r.Patchsets()
	if err != nil {
		return nil, err
	}
	for _, p := range patchsets {
		if p.Name() == name {
			return p, nil
		}
	}
	return nil, nil
}

// Patchsets reads and returns an ordered list of patchsets
func (r *Repo) Patchsets() ([]*patchset.Patchset, error) {
	if len(r.patchsets) == 0 {
		if err := r.walkPatchsets(); err != nil {
			return nil, err
		}
	}
	return r.patchsets, nil
}

// PatchsetMap reads and returns a map of patchset names to patchsets
func (r *Repo) PatchsetMap() (map[string]*patchset.Patchset, error) {
	if len(r.patchsetMap) == 0 {
		if err := r.walkPatchsets(); err != nil {
			return nil, err
		}
	}
	return r.patchsetMap, nil
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
			patchset, err := patchsetFromMetadata(c.Message())
			if err != nil {
				log.Warningf("Error parsing metadata for commit %q: %v", c.Id(), err)
				continue
			}
			if patchset == nil {
				log.Warningf("Got nil patchset for commit %q", c.Id())
				continue
			}
			if _, ok := patchsetMap[patchset.Name()]; ok {
				log.Warningf("Patchset %q seen twice", patchset.Name())
				continue
			}
			patchsets = append(patchsets, patchset)
			patchsetMap[patchset.Name()] = patchset
		}
	}
	r.patchsets = patchsets
	r.patchsetMap = patchsetMap
	return nil
}

func patchsetFromMetadata(metadata string) (*patchset.Patchset, error) {
	fields := parseFields(metadata)
	name, ok := fields[patchsetNameField]
	if !ok {
		return nil, fmt.Errorf("no %s field found", patchsetNameField)
	}
	uuid, ok := fields[patchsetUUIDField]
	if !ok {
		return nil, fmt.Errorf("no %s field found", patchsetUUIDField)
	}
	v, ok := fields[patchsetVersionField]
	if !ok {
		return nil, fmt.Errorf("no %s field found", patchsetVersionField)
	}
	version, err := patchset.ParseVersion(v)
	if err != nil {
		return nil, fmt.Errorf("unable to parse version %q: %w", v, err)
	}
	return patchset.Load(name, uuid, version), nil
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
