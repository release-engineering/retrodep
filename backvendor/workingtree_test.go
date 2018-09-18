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

package backvendor

import (
	"bytes"
	"testing"

	"golang.org/x/tools/go/vcs"
)

func TestStripImportCommentPackage(t *testing.T) {
	src, err := NewGoSource("testdata/godep")
	if err != nil {
		t.Fatal(err)
	}
	wt := &WorkingTree{
		Source: src,
		VCS:    vcs.ByCmd("git"),
	}

	w := bytes.NewBuffer(nil)
	changed, err := wt.StripImportComment("importcomment.go", w)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatalf("changed is incorrect")
	}

	if w.String() != "package foo\n" {
		t.Fatalf("contents incorrect: %v", w.Bytes())
	}
}

func TestStripImportCommentNewline(t *testing.T) {
	src, err := NewGoSource("testdata/godep")
	if err != nil {
		t.Fatal(err)
	}
	wt := &WorkingTree{
		Source: src,
		VCS:    vcs.ByCmd("git"),
	}

	w := bytes.NewBuffer(nil)
	changed, err := wt.StripImportComment("nonl.go", w)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatalf("changed is incorrect")
	}

	b := w.Bytes()
	if b[len(b)-1] != '\n' {
		t.Fatalf("missing newline: %v", w.Bytes())
	}

	w.Reset()
	changed, err = wt.StripImportComment("nl.go", w)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatalf("changed is incorrect")
	}

	w.Reset()
	changed, err = wt.StripImportComment("nonl.txt", w)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatalf("changed is incorrect")
	}
}
