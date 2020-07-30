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

// Package queue defines a marshallable and executable work queue.
package queue

import (
	"errors"
	"fmt"
	"strings"
)

// Operation defines a queueable piece of work.
type Operation struct {
	Name      string
	Execute   func(args []string) error
	Resumable bool
}

// Executor executes a queue of functions corresponding to registered operations.
type Executor struct {
	registered map[string]Operation
	queue      Queue
}

// NewExecutor returns a new, empty Executor.
func NewExecutor() Executor {
	return Executor{
		registered: make(map[string]Operation),
	}
}

// LoadQueue loads a Queue into the executor, appending after any existing queued items.
func (e *Executor) LoadQueue(queue Queue) {
	e.queue.Items = append(e.queue.Items, queue.Items...)
}

// MarshalQueue marshalls the executors operation queue.
func (e *Executor) MarshalQueue() ([]byte, error) {
	return e.queue.MarshalText()
}

// Register registers a new operation with the Executor.
func (e *Executor) Register(op Operation) {
	e.registered[op.Name] = op
}

// Resumable checks whether the named operation is resumable.
func (e *Executor) Resumable(opName string) bool {
	return e.registered[opName].Resumable
}

func (e *Executor) apply(opName string, args []string) error {
	op, ok := e.registered[opName]
	if !ok {
		return fmt.Errorf("apply: invalid operation %q", opName)
	}
	return op.Execute(args)
}

// Execute will execute a single operation from the queue.
func (e *Executor) Execute() error {
	item, err := e.queue.Pop()
	if err != nil {
		return err
	}
	return e.apply(item.Operation, item.Args)
}

// ExecuteAll executes all operations in the queue, stopping on error.
func (e *Executor) ExecuteAll() error {
	var err error
	for err = e.Execute(); err == nil; err = e.Execute() {
	}
	if err != ErrEmpty {
		return err
	}
	return nil
}

// Enqueue queues a new operation with the provided arguments.
func (e *Executor) Enqueue(name string, args ...string) error {
	if _, ok := e.registered[name]; !ok {
		return fmt.Errorf("enqueue: invalid operation %q", name)
	}
	e.queue.Enqueue(name, args...)
	return nil
}

// Peek returns a pointer to the top of the queue.
func (e *Executor) Peek() *Item {
	if len(e.queue.Items) > 0 {
		return &e.queue.Items[0]
	}
	return nil
}

// Queue will return a copy of the operation queue.
func (e *Executor) Queue() Queue {
	return e.queue
}

// Item defines a queued item.
type Item struct {
	Operation string
	Args      []string
}

// ErrEmpty signifies that the queue is empty.
var ErrEmpty = errors.New("no items in queue")

// MarshalText will marshal a byte array representation of an Item.
func (i Item) MarshalText() ([]byte, error) {
	return []byte(strings.Join(append([]string{i.Operation}, i.Args...), " ") + "\n"), nil
}

// UnmarshalText will load the item from the text, overriding any previous values.
func (i *Item) UnmarshalText(text []byte) error {
	s := strings.Fields(string(text))
	if len(s) == 0 {
		return nil
	}
	i.Operation = s[0]
	if len(s) > 1 {
		i.Args = s[1:]
	}
	return nil
}

// Queue defines a queue of operations.
type Queue struct {
	Items []Item
}

// MarshalText will marshal a byte array representation of the queue.
func (q Queue) MarshalText() ([]byte, error) {
	var text []byte
	for _, i := range q.Items {
		bytes, err := i.MarshalText()
		if err != nil {
			return nil, err
		}
		text = append(text, bytes...)
	}
	return text, nil
}

// UnmarshalText will load the queue with the items from the text, appending them to the existing items.
func (q *Queue) UnmarshalText(text []byte) error {
	ss := strings.Split(string(text), "\n")
	for _, s := range ss {
		i := Item{}
		err := i.UnmarshalText([]byte(s))
		if err != nil {
			return err
		}
		if i.Operation == "" {
			continue
		}
		q.Items = append(q.Items, i)
	}
	return nil
}

// Enqueue will add the operation and its arguments to the queue.
func (q *Queue) Enqueue(name string, args ...string) {
	q.Items = append(q.Items, Item{
		Operation: name,
		Args:      args,
	})
}

// Pop will remove a single item from the queue, or return ErrEmpty.
func (q *Queue) Pop() (Item, error) {
	if len(q.Items) < 1 {
		return Item{}, ErrEmpty
	}
	item := q.Items[0]
	q.Items = q.Items[1:]
	return item, nil
}
