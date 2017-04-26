// Copyright 2017 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !cov

package e2e

import "github.com/coreos/etcd/pkg/expect"

const noOutputLineCount = 0 // regular binaries emit no extra lines

func spawnCmd(args []string) (*expect.ExpectProcess, error) {
	return expect.NewExpect(args[0], args[1:]...)
}
