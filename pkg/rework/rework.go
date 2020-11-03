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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/golang/glog"
	"github.com/google/kilt/pkg/dependency"
	"github.com/google/kilt/pkg/patchset"
	"github.com/google/kilt/pkg/queue"
	"github.com/google/kilt/pkg/repo"
)

// Command defines a rework command.
type Command struct {
	repo     *repo.Repo
	executor queue.Executor
	writer   stateWriter
	reader   stateReader
}

// NewCommand opens the repo and returns a new rework command.
func NewCommand() (*Command, error) {
	r, err := repo.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rework: %w", err)
	}
	e := queue.NewExecutor()
	var state *stateFile
	return &Command{
		repo:     r,
		executor: e,
		writer:   state,
		reader:   state,
	}, nil
}

func (c *Command) setWriter(w stateWriter) {
	c.writer = w
}

func (c *Command) setReader(r stateReader) {
	c.reader = r
}

// Save will marshal and save the command. Currently a placeholder that just prints it.
func (c *Command) Save() error {
	return c.writer.WriteQueueState(c.executor.Queue())
}

// Execute will execute the command, running an queued operations.
func (c *Command) Execute() error {
	item := c.executor.Peek()
	if item != nil && c.executor.Resumable(item.Operation) {
		if err := c.writer.WriteCurrentState(*item); err != nil {
			return err
		}
	}
	err := c.executor.Execute()
	if err == nil {
		return c.writer.ClearCurrentState()
	}
	return err
}

// ExecuteAll will execute all queued operations, stopping if an error occurs.
func (c *Command) ExecuteAll() error {
	var err error
	for err = c.Execute(); err == nil; err = c.Execute() {
	}
	if err == queue.ErrEmpty {
		return nil
	}
	return err
}

// stateWriter manages the writing and removal of operation states.
type stateWriter interface {
	WriteQueueState(queue queue.Queue) error
	WriteCurrentState(item queue.Item) error
	ClearQueueState() error
	ClearCurrentState() error
}

// stateReader manages the reading of operation states.
type stateReader interface {
	ReadState() (queue.Queue, error)
	ReadCurrentState() (queue.Queue, error)
}

type stateFile struct {
	path, name string
}

// ReadState will read the operation queue, returning a new Queue.
func (s *stateFile) ReadState() (queue.Queue, error) {
	var q queue.Queue
	if s == nil {
		return q, nil
	}
	var e *os.PathError
	file, err := ioutil.ReadFile(filepath.Join(s.path, s.name))
	if errors.As(err, &e) {
		return q, nil
	} else if err != nil {
		return q, err
	}
	err = q.UnmarshalText(file)
	if err != nil {
		return q, err
	}
	return q, nil
}

// ReadState will read the current operation, returning a new Queue.
func (s *stateFile) ReadCurrentState() (queue.Queue, error) {
	var item queue.Item
	var queue queue.Queue
	if s == nil {
		return queue, nil
	}
	file, err := ioutil.ReadFile(filepath.Join(s.path, s.name+"-current"))
	var e *os.PathError
	if err != nil && !errors.As(err, &e) {
		return queue, err
	}
	if len(file) == 0 {
		return queue, nil
	}
	err = item.UnmarshalText(file)
	if err != nil {
		return queue, err
	}
	if item.Operation == "" {
		return queue, nil
	}
	queue.Items = append(queue.Items, item)
	return queue, nil
}

// WriteCurrentState will write the current item to a state file.
func (s *stateFile) WriteCurrentState(item queue.Item) error {
	if s == nil {
		return nil
	}
	if item.Operation == "" {
		return s.ClearCurrentState()
	}
	os.MkdirAll(s.path, 0777)
	currentFile := filepath.Join(s.path, s.name+"-current")
	text, err := item.MarshalText()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(currentFile, text, 0666)
}

// WriteQueueState will marshal and write the queue to a state file.
func (s *stateFile) WriteQueueState(queue queue.Queue) error {
	if s == nil {
		return nil
	}
	if len(queue.Items) == 0 {
		return s.ClearQueueState()
	}
	os.MkdirAll(s.path, 0777)
	q, err := queue.MarshalText()
	if err != nil {
		return fmt.Errorf("failed to marshal queue: %v", err)
	}
	queueFile := filepath.Join(s.path, s.name)
	return ioutil.WriteFile(queueFile, q, 0666)
}

// ClearCurrentState will remove the current operation state file.
func (s *stateFile) ClearCurrentState() error {
	if s == nil {
		return nil
	}
	queueFile := filepath.Join(s.path, s.name)
	return os.RemoveAll(queueFile + "-current")
}

// ClearCurrentState will remove the queue state file.
func (s *stateFile) ClearQueueState() error {
	if s == nil {
		return nil
	}
	queueFile := filepath.Join(s.path, s.name)
	return os.RemoveAll(queueFile)
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

func registerBuildOperations(e *queue.Executor, r *repo.Repo) {
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
			Name: "Finish",
			Execute: func(branch []string) error {
				if len(branch) == 0 {
					return errors.New("no branch specified")
				}
				return finishBuild(r, branch[0])
			},
		},
		{
			Name: "Abort",
			Execute: func(_ []string) error {
				return abortRework(r)
			},
		},
		{
			Name: "Begin",
			Execute: func(_ []string) error {
				return startNewRework(r)
			},
		},
		{
			Name: "Checkout",
			Execute: func(revspec []string) error {
				if len(revspec) == 0 {
					return errors.New("no rev specified")
				}
				fmt.Printf("Checking out %s\n", revspec[0])
				return r.CheckoutRev(revspec[0])
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
			Name: "Abort",
			Execute: func(_ []string) error {
				return abortRework(r)
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
			Name: "Skip",
			Execute: func([]string) error {
				fmt.Println("Clearing queue")
				return skipReworkQueue(r)
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

	s := newStateFile(c.repo, "queue")

	c.setWriter(s)
	c.setReader(s)

	registerOperations(&c.executor, c.repo)

	if exists, err := c.repo.ReworkInProgress(); err != nil {
		return nil, err
	} else if exists {
		if q, err := c.reader.ReadState(); err == nil && len(q.Items) > 0 {
			return nil, fmt.Errorf("rework already in progress")
		}
	} else {
		if err = c.executor.Enqueue("Begin"); err != nil {
			return nil, err
		}
	}
	patchsets, err := c.repo.Patchsets()
	if err != nil {
		return nil, err
	}
	revDeps, err := selectRevDepPatchsets(c.repo, selectors)
	if err != nil {
		return nil, err
	}
	first := true
	var previous *patchset.Patchset
	i := 0
	for _, p := range patchsets {
		if i < len(revDeps) && revDeps[i].SameAs(p) {
			if first {
				if previous != nil {
					c.executor.Enqueue("Checkout", previous.Name())
				} else {
					c.executor.Enqueue("CheckoutBase")
				}
				first = false
			}
			c.executor.Enqueue("Rework", p.Name())
			i++
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

func selectRevDepPatchsets(r *repo.Repo, selectors []TargetSelector) ([]*patchset.Patchset, error) {
	patchsets, err := r.PatchsetCache()
	if err != nil {
		return nil, err
	}
	deps := dependency.NewStruct(patchsets)
	b, err := ioutil.ReadFile("dependencies.json")
	if err != nil {
		log.Exitf(`Failed to read "dependencies.json": %v`, err)
	}
	err = json.Unmarshal(b, deps)
	if err != nil {
		log.Exitf(`Failed to load "dependencies.json": %v`, err)
	}
	seen := map[string]struct{}{}
	var selected []*patchset.Patchset
	for _, p := range patchsets.Slice {
		for _, s := range selectors {
			if _, ok := seen[p.Name()]; !ok && s.Select(p) {
				seen[p.Name()] = struct{}{}
				selected = append(selected, p)
				ps := deps.TransitiveReverseDependencies(p)
				for _, patchset := range ps {
					seen[patchset.Name()] = struct{}{}
				}
				selected = append(selected, ps...)
			}
		}
	}
	sort.Slice(selected, func(i, j int) bool {
		return patchsets.Index[selected[i].Name()] < patchsets.Index[selected[j].Name()]
	})
	return selected, err
}

// NewBeginBuildCommand returns a command that begins a new rework.
func NewBeginBuildCommand(base string, selectors ...TargetSelector) (*Command, error) {
	c, err := NewCommand()
	if err != nil {
		return nil, err
	}

	s := newStateFile(c.repo, "queue")

	c.setWriter(s)
	c.setReader(s)

	registerBuildOperations(&c.executor, c.repo)

	if err = c.executor.Enqueue("Begin"); err != nil {
		return nil, err
	}
	selected, err := selectDependentPatchsets(c.repo, selectors)
	if err != nil {
		return nil, err
	}
	if err = c.executor.Enqueue("Checkout", base); err != nil {
		return nil, err
	}
	for _, p := range selected {
		if err = c.executor.Enqueue("Apply", p.Name()); err != nil {
			return nil, err
		}
	}
	if err = c.executor.Enqueue("UpdateHead"); err != nil {
		return nil, err
	}
	if err = c.executor.Enqueue("Finish", base); err != nil {
		return nil, err
	}
	return c, nil
}

func selectDependentPatchsets(r *repo.Repo, selectors []TargetSelector) ([]*patchset.Patchset, error) {
	patchsets, err := r.PatchsetCache()
	if err != nil {
		return nil, err
	}
	deps := dependency.NewStruct(patchsets)
	b, err := ioutil.ReadFile("dependencies.json")
	if err != nil {
		log.Exitf(`Failed to read "dependencies.json": %v`, err)
	}
	err = json.Unmarshal(b, deps)
	if err != nil {
		log.Exitf(`Failed to load "dependencies.json": %v`, err)
	}
	seen := map[string]struct{}{}
	var selected []*patchset.Patchset
	for _, p := range patchsets.Slice {
		for _, s := range selectors {
			if _, ok := seen[p.Name()]; !ok && s.Select(p) {
				seen[p.Name()] = struct{}{}
				selected = append(selected, p)
				ps := deps.TransitiveDependencies(p)
				for _, patchset := range ps {
					seen[patchset.Name()] = struct{}{}
				}
				selected = append(selected, ps...)
			}
		}
	}
	sort.Slice(selected, func(i, j int) bool {
		return patchsets.Index[selected[i].Name()] < patchsets.Index[selected[j].Name()]
	})
	return selected, err
}

func startNewBuild(r *repo.Repo, branch string) error {
	if exists, err := r.ReworkInProgress(); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("rework already in progress")
	}
	if err := r.WriteRefHead("rework/head"); err != nil {
		return err
	}
	if err := r.WriteSymbolicRefBranch("rework/branch", branch); err != nil {
		return err
	}
	return r.SetHead("rework/head")
}

func startNewRework(r *repo.Repo) error {
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
	if exists, err := c.repo.ReworkInProgress(); err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("no rework in progress")
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

func finishBuild(r *repo.Repo, branch string) error {
	if exists, err := r.ReworkInProgress(); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("no rework in progress")
	}
	if err := r.SetBranchToHead(branch); err != nil {
		return err
	}
	if err := r.CheckoutBranch(branch); err != nil {
		return err
	}
	cleanupReworkState(r)
	return nil
}

func finishRework(r *repo.Repo) error {
	if err := r.SetIndirectBranchToHead("rework/branch"); err != nil {
		return err
	}
	if err := r.CheckoutIndirectBranch("rework/branch"); err != nil {
		return err
	}
	cleanupReworkState(r)
	return nil
}

// NewAbortCommand returns a command that aborts an in-progress rework.
func NewAbortCommand() (*Command, error) {
	c, err := NewCommand()
	if err != nil {
		return nil, err
	}
	s := newStateFile(c.repo, "queue")

	c.setWriter(s)
	if exists, err := c.repo.ReworkInProgress(); err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("no rework in progress")
	}
	registerOperations(&c.executor, c.repo)
	if err = c.executor.Enqueue("Abort"); err != nil {
		return nil, err
	}
	return c, nil
}

func abortRework(r *repo.Repo) error {
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
	if exists, err := c.repo.ReworkInProgress(); err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("no rework in progress")
	}
	registerOperations(&c.executor, c.repo)
	if err = c.executor.Enqueue("Validate"); err != nil {
		return nil, err
	}
	return c, nil
}

func validateRework(r *repo.Repo) (bool, error) {
	return r.CompareTreeToHead("rework/branch")
}

func newStateFile(r *repo.Repo, name string) *stateFile {
	return &stateFile{
		path: filepath.Join(r.KiltDirectory(), "rework"),
		name: name,
	}
}

// Status prints the status of the rework.
func Status(r *repo.Repo) error {
	state := newStateFile(r, "queue")
	q, err := state.ReadState()
	if err != nil {
		return err
	}
	if len(q.Items) > 0 {
		fmt.Println("Remaining work:")
		for _, item := range q.Items {
			fmt.Printf("\t%s %s\n", item.Operation, strings.Join(item.Args, " "))
		}
		fmt.Println(`Use kilt rework --continue to perform the next operation, or manually perform
the operation and use kilt rework --skip to skip execution.`)
	} else {
		fmt.Println("All work complete. Use kilt rework --finish to validate and finish the rework.")
	}
	return nil
}

// NewContinueCommand returns a command that continues with saved rework steps.
func NewContinueCommand() (*Command, error) {
	c, err := NewCommand()
	if err != nil {
		return nil, err
	}

	state := newStateFile(c.repo, "queue")
	c.setWriter(state)
	c.setReader(state)

	if exists, err := c.repo.ReworkInProgress(); err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("no rework in progress")
	}

	registerOperations(&c.executor, c.repo)

	if err = continueRework(c); err != nil {
		return nil, err
	}

	return c, nil
}

// NewSkipCommand returns a command that skips the next saved rework step.
func NewSkipCommand() (*Command, error) {
	c, err := NewCommand()
	if err != nil {
		return nil, err
	}

	state := newStateFile(c.repo, "queue")
	c.setWriter(state)
	c.setReader(state)

	if exists, err := c.repo.ReworkInProgress(); err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("no rework in progress")
	}

	registerOperations(&c.executor, c.repo)

	c.executor.Enqueue("Skip")

	q, err := c.reader.ReadState()
	if err != nil {
		return nil, err
	}
	current, err := c.reader.ReadCurrentState()
	if err != nil {
		return nil, err
	}
	if len(current.Items) == 0 {
		if _, err = q.Pop(); err != nil {
			return nil, err
		}
	} else {
		c.writer.ClearCurrentState()
	}
	c.executor.LoadQueue(q)

	return c, nil
}

func continueRework(c *Command) error {
	current, err := c.reader.ReadCurrentState()
	if err != nil {
		return err
	}
	c.executor.LoadQueue(current)
	q, err := c.reader.ReadState()
	if err != nil {
		return err
	}
	c.executor.LoadQueue(q)
	return nil
}

func skipReworkQueue(r *repo.Repo) error {
	state := newStateFile(r, "reworkQueue")
	if err := state.ClearQueueState(); err != nil {
		return err
	}
	return state.ClearCurrentState()
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
	state := newStateFile(r, "reworkQueue")
	c.setWriter(state)
	c.setReader(state)

	registerReworkOperations(&c.executor, c.repo)

	current, err := c.reader.ReadCurrentState()
	if err != nil {
		return err
	}
	q, err := c.reader.ReadState()
	if err != nil {
		return err
	}
	c.executor.LoadQueue(q)

	if len(q.Items) == 0 && len(current.Items) == 0 {
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
	}
	if err = c.ExecuteAll(); err != nil {
		if saveErr := c.Save(); saveErr != nil {
			return fmt.Errorf("failed to save queue: %v; during error: %v", saveErr, err)
		}
		return err
	}
	return nil
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
	state := newStateFile(r, "reworkQueue")
	c.setWriter(state)
	c.setReader(state)

	registerReworkOperations(&c.executor, c.repo)

	current, err := c.reader.ReadCurrentState()
	if err != nil {
		return err
	}
	q, err := c.reader.ReadState()
	if err != nil {
		return err
	}
	c.executor.LoadQueue(q)

	if len(q.Items) == 0 && len(current.Items) == 0 {
		c.executor.Enqueue("Apply", p.MetadataCommit())
		for _, patch := range p.Patches() {
			c.executor.Enqueue("Apply", patch)
		}
	}
	if err = c.ExecuteAll(); err != nil {
		if saveErr := c.Save(); saveErr != nil {
			return fmt.Errorf("failed to save queue: %v; during error: %v", saveErr, err)
		}
		return err
	}
	return nil
}

func registerReworkOperations(e *queue.Executor, r *repo.Repo) {
	var operations = []queue.Operation{
		{
			Name: "Apply",
			Execute: func(patch []string) error {
				desc, err := r.DescribeCommit(patch[0])
				if err != nil {
					return err
				}
				fmt.Printf("Applying %s\n", desc)
				return r.CherryPickToHead(patch[0])
			},
			Resumable: true,
		},
		{
			Name: "Cherrypick",
			Execute: func(patch []string) error {
				desc, err := r.DescribeCommit(patch[0])
				if err != nil {
					return err
				}
				fmt.Printf("Cherrypick %s\n", desc)
				return r.CherryPickToHead(patch[0])
			},
			Resumable: true,
		},
		{
			Name: "UpdateMetadata",
			Execute: func(patch []string) error {
				desc, err := r.DescribeCommit(patch[0])
				if err != nil {
					return err
				}
				fmt.Printf("Updating metadata %s\n", desc)
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
