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
	"fmt"

	log "github.com/golang/glog"
	"github.com/google/kilt/pkg/queue"
	"github.com/google/kilt/pkg/repo"
)

// Init initializes a new rework. Currently just a placeholder with a fake operation.
func Init() (queue.Executor, error) {
	if _, err := repo.Open(); err != nil {
		return queue.Executor{}, fmt.Errorf("failed to initialize rework: %w", err)
	}
	e := queue.NewExecutor()
	e.Register(queue.Operation{
		Name: "asdf",
		Execute: func(args []string) error {
			log.Info("test")
			return nil
		},
	})
	return e, e.Enqueue("asdf", "arg")
}
