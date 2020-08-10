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

// Package rework defines rework commands and managing rework state.
package rework

import (
	"errors"
	"fmt"

	log "github.com/golang/glog"
	"github.com/google/kilt/pkg/patchset"
	"github.com/google/kilt/pkg/queue"
	"github.com/google/kilt/pkg/repo"
)

// Command defines a rework command.
type Command struct {
	repo     *repo.Repo
	executor queue.Executor
}

// NewCommand opens the repo and returns a new rework command.
func NewCommand() (*Command, error) {
	r, err := repo.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rework: %w", err)
	}
	e := queue.NewExecutor()
	return &Command{repo: r, executor: e}, nil
}

// Save will marshal and save the command. Currently a placeholder that just prints it.
func (c Command) Save() {
	q, err := c.executor.MarshalQueue()
	if err == nil {
		fmt.Println(string(q))
	}
}

// Execute will execute the command, running an queued operations.
func (c *Command) Execute() error {
	return c.executor.ExecuteAll()
}

// TargetSelector selects patchsets based on some criteria.
type TargetSelector interface {
	Select(patchset *patchset.Patchset) bool
}

// FloatingTargets selects patchsets that have any floating targets.
type FloatingTargets struct{}

// Select will return true if the patchset has any floating targets.
func (FloatingTargets) Select(patchset *patchset.Patchset) bool {
	return len(patchset.FloatingPatches()) > 0
}

// AllTargets selects every patchset.
type AllTargets struct{}

// Select will always return true.
func (AllTargets) Select(_ *patchset.Patchset) bool {
	return true
}

// NoTargets selects nothing.
type NoTargets struct{}

// Select will always return false.
func (NoTargets) Select(_ *patchset.Patchset) bool {
	return false
}

// PatchsetTarget selects a specified patchset.
type PatchsetTarget struct {
	Name string
}

// Select returns true if the patchset name matches.
func (t PatchsetTarget) Select(patchset *patchset.Patchset) bool {
	return t.Name == patchset.Name()
}

func registerOperations(e *queue.Executor, r *repo.Repo) {
	var operations = []queue.Operation{
		{
			Name: "UpdateHead",
			Execute: func(_ []string) error {
				if err := r.WriteRefHead("rework/head"); err != nil {
					return err
				}
				return r.SetHead("rework/head")
			},
		},
		{
			Name: "Validate",
			Execute: func(_ []string) error {
				if valid, err := validateRework(r); err != nil {
					return err
				} else if !valid {
					return &ErrInvalidRework{
						original: "refs/kilt/rework/branch",
						reworked: "HEAD",
					}
				}
				return nil
			},
		},
		{
			Name: "Finish",
			Execute: func(_ []string) error {
				return finishRework(r)
			},
		},
		{
			Name: "Begin",
			Execute: func(_ []string) error {
				return startNewRework(r)
			},
		},
		{
			Name: "Rework",
			Execute: func(patchset []string) error {
				if len(patchset) == 0 {
					return errors.New("no patchset specified")
				}
				fmt.Printf("Reworking patchset %s\n", patchset[0])
				return reworkPatchset(r, patchset[0])
			},
			Resumable: true,
		},
		{
			Name: "Checkout",
			Execute: func(patchset []string) error {
				if len(patchset) == 0 {
					return errors.New("no patchset specified")
				}
				fmt.Printf("Checking out patchset %s\n", patchset[0])
				return r.CheckoutPatchset(patchset[0])
			},
			Resumable: true,
		},
		{
			Name: "CheckoutBase",
			Execute: func(patchset []string) error {
				fmt.Println("Checking out kilt base")
				return r.CheckoutBase()
			},
			Resumable: true,
		},
		{
			Name: "Apply",
			Execute: func(patchset []string) error {
				if len(patchset) == 0 {
					return errors.New("no patchset specified")
				}
				fmt.Printf("Applying patchset %s\n", patchset[0])
				return applyPatchset(r, patchset[0])
			},
			Resumable: true,
		},
	}
	for _, op := range operations {
		e.Register(op)
	}
}

func selectPatchset(selectors []TargetSelector, patchset *patchset.Patchset) bool {
	for _, s := range selectors {
		if s.Select(patchset) {
			return true
		}
	}
	return false
}

// NewBeginCommand returns a command that begins a new rework.
func NewBeginCommand(selectors ...TargetSelector) (*Command, error) {
	c, err := NewCommand()
	if err != nil {
		return nil, err
	}
	registerOperations(&c.executor, c.repo)

	if err = c.executor.Enqueue("Begin"); err != nil {
		return nil, err
	}
	patchsets, err := c.repo.Patchsets()
	if err != nil {
		return nil, err
	}
	first := true
	var previous *patchset.Patchset
	for _, p := range patchsets {
		if selectPatchset(selectors, p) {
			if first {
				if previous != nil {
					c.executor.Enqueue("Checkout", previous.Name())
				} else {
					c.executor.Enqueue("CheckoutBase")
				}
				first = false
			}
			c.executor.Enqueue("Rework", p.Name())
		} else {
			if !first {
				c.executor.Enqueue("Apply", p.Name())
			} else {
				previous = p
			}
		}
	}
	if err = c.executor.Enqueue("UpdateHead"); err != nil {
		return nil, err
	}
	return c, nil
}

func startNewRework(r *repo.Repo) error {
	if exists, err := r.ReworkInProgress(); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("rework already in progress")
	}
	if err := r.WriteRefHead("rework/head"); err != nil {
		return err
	}
	if err := r.WriteSymbolicRefHead("rework/branch"); err != nil {
		return err
	}
	return r.SetHead("rework/head")
}

// NewFinishCommand returns a command that finishes a rework.
func NewFinishCommand(force bool) (*Command, error) {
	c, err := NewCommand()
	if err != nil {
		return nil, err
	}
	registerOperations(&c.executor, c.repo)
	if !force {
		if err = c.executor.Enqueue("Validate"); err != nil {
			return nil, err
		}
	}
	if err = c.executor.Enqueue("Finish"); err != nil {
		return nil, err
	}
	return c, nil
}

func finishRework(r *repo.Repo) error {
	if exists, err := r.ReworkInProgress(); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("no rework in progress")
	}
	if err := r.SetIndirectBranchToHead("rework/branch"); err != nil {
		return err
	}
	if err := r.CheckoutIndirectBranch("rework/branch"); err != nil {
		return err
	}
	cleanupReworkState(r)
	return nil
}

// ErrInvalidRework indicates that the rework is invalid and the trees don't match.
type ErrInvalidRework struct {
	original, reworked string
}

func (e *ErrInvalidRework) Error() string {
	return fmt.Sprintf("rework tree doesn't match: git diff-tree -p %s %s", e.original, e.reworked)
}

// NewValidateCommand returns a command that checks the validity of the rework.
func NewValidateCommand() (*Command, error) {
	c, err := NewCommand()
	if err != nil {
		return nil, err
	}
	registerOperations(&c.executor, c.repo)
	if err = c.executor.Enqueue("Validate"); err != nil {
		return nil, err
	}
	return c, nil
}

func validateRework(r *repo.Repo) (bool, error) {
	if exists, err := r.ReworkInProgress(); err != nil {
		return false, err
	} else if !exists {
		return false, fmt.Errorf("no rework in progress")
	}
	return r.CompareTreeToHead("rework/branch")
}

func reworkPatchset(r *repo.Repo, patchset string) error {
	patchsets, err := r.PatchsetMap()
	if err != nil {
		return err
	}
	p, ok := patchsets[patchset]
	if !ok {
		return fmt.Errorf("patchset %q not found", patchset)
	}
	c, err := NewCommand()
	if err != nil {
		return err
	}

	registerReworkOperations(&c.executor, c.repo)

	if p.MetadataCommit() == "" {
		c.executor.Enqueue("CreateMetadata", p.Name())
	} else {
		c.executor.Enqueue("UpdateMetadata", p.MetadataCommit())
	}
	for _, patch := range p.Patches() {
		c.executor.Enqueue("Apply", patch)
	}
	for _, patch := range p.FloatingPatches() {
		c.executor.Enqueue("Cherrypick", patch)
	}
	return c.Execute()
}

func applyPatchset(r *repo.Repo, patchset string) error {
	patchsets, err := r.PatchsetMap()
	if err != nil {
		return err
	}
	p, ok := patchsets[patchset]
	if !ok {
		return fmt.Errorf("patchset %q not found", patchset)
	}
	c, err := NewCommand()
	if err != nil {
		return err
	}

	registerReworkOperations(&c.executor, c.repo)

	c.executor.Enqueue("Apply", p.MetadataCommit())
	for _, patch := range p.Patches() {
		c.executor.Enqueue("Apply", patch)
	}
	return c.Execute()
}

func registerReworkOperations(e *queue.Executor, r *repo.Repo) {
	var operations = []queue.Operation{
		{
			Name: "Apply",
			Execute: func(patch []string) error {
				fmt.Printf("Applying %s\n", patch[0])
				return r.CherryPickToHead(patch[0])
			},
			Resumable: true,
		},
		{
			Name: "Cherrypick",
			Execute: func(patch []string) error {
				fmt.Printf("Cherrypick %s\n", patch[0])
				return r.CherryPickToHead(patch[0])
			},
			Resumable: true,
		},
		{
			Name: "UpdateMetadata",
			Execute: func(patch []string) error {
				fmt.Printf("Updating metadata %s\n", patch[0])
				return r.UpdateMetadataForCommit(patch[0])
			},
			Resumable: true,
		},
		{
			Name: "CreateMetadata",
			Execute: func(ps []string) error {
				fmt.Printf("Creating metadata for %s\n", ps[0])
				p := patchset.New(ps[0])
				return r.AddPatchset(p)
			},
			Resumable: true,
		},
	}
	for _, op := range operations {
		e.Register(op)
	}
}

func cleanupReworkState(r *repo.Repo) {
	if err := r.DeleteKiltRef("rework/branch"); err != nil {
		log.Errorf("Error deleting kilt rework branch ref: %v", err)
	}
	if err := r.DeleteKiltRef("rework/head"); err != nil {
		log.Errorf("Error deleting kilt rework head ref: %v", err)
	}
}

type reworkState struct {
	branch, head string
}

func loadExistingRework(r *repo.Repo) (reworkState, error) {
	var head, branch string
	if branch, err := r.LookupKiltRef("rework/branch"); err != nil {
		return reworkState{}, err
	} else if branch == "" {
		return reworkState{}, errors.New("failed to lookup rework branch")
	}
	if head, err := r.LookupKiltRef("rework/head"); err != nil {
		return reworkState{}, err
	} else if head == "" {
		return reworkState{}, errors.New("failed to lookup rework head")
	}
	return reworkState{branch: branch, head: head}, nil
}
