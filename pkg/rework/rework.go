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

func registerOperations(e *queue.Executor, r *repo.Repo) {
	var operations = []queue.Operation{
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
	}
	for _, op := range operations {
		e.Register(op)
	}
}

// NewBeginCommand returns a command that begins a new rework.
func NewBeginCommand() (*Command, error) {
	c, err := NewCommand()
	if err != nil {
		return nil, err
	}
	registerOperations(&c.executor, c.repo)

	if err = c.executor.Enqueue("Begin"); err != nil {
		return nil, err
	}
	return c, nil
}

func startNewRework(r *repo.Repo) error {
	if exists, err := checkExistingRework(r); err != nil {
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
func NewFinishCommand() (*Command, error) {
	c, err := NewCommand()
	if err != nil {
		return nil, err
	}
	registerOperations(&c.executor, c.repo)
	if err = c.executor.Enqueue("Finish"); err != nil {
		return nil, err
	}
	return c, nil
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

func checkExistingRework(r *repo.Repo) (bool, error) {
	if s, err := r.LookupKiltRef("rework/branch"); err != nil {
		return false, err
	} else if s != "" {
		return true, nil
	}
	if s, err := r.LookupKiltRef("rework/head"); err != nil {
		return false, err
	} else if s != "" {
		return true, nil
	}
	return false, nil
}
