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

	"github.com/release-engineering/backvendor/backvendor"
)

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Printf("Usage: %s path\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}
	src := backvendor.GoSource{
		Path: flag.Arg(0),
	}
	vendored, err := src.VendoredProjects()
	if err != nil {
		log.Fatal(err)
	}
	for _, project := range vendored {
		vp, err := src.DescribeVendoredProject(project)
		if err != nil {
			if err == backvendor.ErrorVersionNotFound {
				fmt.Printf("%s: ?\n", project.Root)
				continue
			}
			log.Fatalf("%s: %s\n", project.Root, err)
		}
		fmt.Printf("%s: %s\n", project.Root, *vp)
	}
}
