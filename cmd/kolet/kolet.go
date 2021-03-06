// Copyright 2015 CoreOS, Inc.
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

package main

import (
	"fmt"
	"os"

	"github.com/coreos/mantle/Godeps/_workspace/src/github.com/spf13/cobra"
	"github.com/coreos/mantle/cli"
	"github.com/coreos/mantle/kola"
)

var (
	root = &cobra.Command{
		Use:   "kolet [command]",
		Short: "Native code runner for kola",
	}

	cmdRun = &cobra.Command{
		Use:   "run <test name> <func name>",
		Short: "Run native tests a group at a time",
		Run:   Run,
	}
)

func main() {
	root.AddCommand(cmdRun)
	cli.Execute(root)
}

// test runner
func Run(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "kolet: Extra arguements specified. Usage: 'kolet run <test name> <func name>'\n")
		os.Exit(2)
	}
	testname, funcname := args[0], args[1]

	// find test with matching name
	test, ok := kola.Tests[testname]
	if !ok {
		fmt.Fprintf(os.Stderr, "kolet: test group not found\n")
		os.Exit(1)
	}
	// find native function in test
	f, ok := test.NativeFuncs[funcname]
	if !ok {
		fmt.Fprintf(os.Stderr, "kolet: native function not found\n")
		os.Exit(1)
	}
	err := f()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kolet: on native test %v: %v", funcname, err)
		os.Exit(1)
	}
}
