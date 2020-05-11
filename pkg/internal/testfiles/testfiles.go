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

// Package testfiles is a testhelper that abstracts the path to test files.
package testfiles

import (
	"fmt"
	"io/ioutil"
)

// TempDir creates and returns the full path to a temp test dir
func TempDir(name string) (string, error) {
	dir, err := ioutil.TempDir("", fmt.Sprintf("kilt-test-%s", name))
	return dir, err
}
