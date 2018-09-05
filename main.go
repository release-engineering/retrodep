// Copyright (C) 2018 Tim Waugh
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/release-engineering/backvendor/backvendor"
)

var helpFlag = flag.Bool("help", false, "print help")
var importPath = flag.String("importpath", "", "top-level import path")
var depsFlag = flag.Bool("deps", true, "show vendored dependencies")

func display(name string, ref *backvendor.Reference) {
	fmt.Printf("%s", name)
	if ref.Rev != "" {
		fmt.Printf("@%s", ref.Rev)
	}
	if ref.Tag != "" {
		fmt.Printf(" =%s", ref.Tag)
	}
	if ref.Ver != "" {
		fmt.Printf(" ~%s", ref.Ver)
	}
	fmt.Printf("\n")
}

func showTopLevel(src *backvendor.GoSource) {
	main, err := src.Project(*importPath)
	if err != nil {
		if err == backvendor.ErrorNeedImportPath {
			log.Printf("%s: %s", src.Path, err)
			fmt.Fprintln(os.Stderr,
				"Provide import path with -importpath")
			os.Exit(1)
		}
		log.Fatalf("%s: %s", src.Path, err)
	}

	project, err := backvendor.DescribeProject(main, src.Path)
	switch err {
	case backvendor.ErrorVersionNotFound:
		fmt.Printf("*%s ?\n", main.Root)
	case nil:
		display("*"+main.Root, project)
	default:
		log.Fatalf("%s: %s", src.Path, err)
	}
}

func showVendored(src *backvendor.GoSource) {
	vendored, err := src.VendoredProjects()
	if err != nil {
		log.Fatal(err)
	}

	// Sort the projects for predictable output
	var repos []string
	for repo := range vendored {
		repos = append(repos, repo)
	}
	sort.Strings(repos)

	// Describe each vendored project
	for _, repo := range repos {
		project := vendored[repo]
		vp, err := src.DescribeVendoredProject(project)
		switch err {
		case backvendor.ErrorVersionNotFound:
			fmt.Printf("%s ?\n", project.Root)
		case nil:
			display(project.Root, vp)
		default:
			log.Fatalf("%s: %s\n", project.Root, err)
		}
	}
}

func processArgs(args []string) *backvendor.GoSource {
	progName := filepath.Base(args[0])
	log.SetFlags(0) // For typical stderr output of a program.

	// Stop the default behaviour of printing errors and exiting.
	// Instead, silence the printing and return them.
	cli := flag.CommandLine
	cli.Init("", flag.ContinueOnError)
	cli.SetOutput(ioutil.Discard)
	cli.Usage = func() {}

	usageMsg := fmt.Sprintf("usage: %s [-help] [-importpath=toplevel] [-deps=false] path", progName)
	usage := func(flaw string) {
		log.Fatalf("%s: %s\n%s\n", progName, flaw, usageMsg)
	}
	err := cli.Parse(args[1:])
	if err == flag.ErrHelp || *helpFlag { // Handle ‘-h’.
		fmt.Printf("%s: help requested\n%s\n", progName, usageMsg)
		cli.SetOutput(os.Stdout)
		flag.PrintDefaults()
		os.Exit(0) // Not an error.
	}
	if err != nil {
		usage(err.Error())
	}

	narg := flag.NArg()
	if narg == 0 {
		usage("missing path")
	}
	if narg != 1 {
		usage(fmt.Sprintf("only one path allowed: %q", flag.Arg(1)))
	}

	return backvendor.NewGoSource(flag.Arg(0))
}

func main() {
	src := processArgs(os.Args)
	showTopLevel(src)
	if *depsFlag {
		showVendored(src)
	}
}
