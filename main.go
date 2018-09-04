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
	"log"
	"os"
	"sort"

	"github.com/release-engineering/backvendor/backvendor"
)

var importPath = flag.String("importpath", "", "top-level import path")

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

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Printf("Usage: %s path\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}
	src := backvendor.GoSource(flag.Arg(0))

	main, err := src.Project(*importPath)
	if err != nil {
		log.Fatalf("%s: %s", src.Topdir(), err)
	}

	project, err := backvendor.DescribeProject(main, src.Topdir())
	switch err {
	case backvendor.ErrorVersionNotFound:
		fmt.Printf("*%s ?\n", main.Root)
	case nil:
		display("*"+main.Root, project)
	default:
		log.Fatalf("%s: %s", src.Topdir(), err)
	}

	vendored, err := src.VendoredProjects()
	if err != nil {
		log.Fatal(err)
	}

	// Sort the projects for predictable output
	var repos []string
	for repo, _ := range vendored {
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
