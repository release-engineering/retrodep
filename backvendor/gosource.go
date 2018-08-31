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
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

// GoSource represents a filesystem tree containing Go source code.
type GoSource struct {
	// Path is the top-level path of the filesystem tree.
	Path string

	// vendor is the path to the vendored source code.
	vendor string

	// pkgs contains information about the packages and their dependencies.
	pkgs []*packages.Package
}

// Vendor returns the path to the vendored source code.
func (src GoSource) Vendor() string {
	if src.vendor == "" {
		src.vendor = filepath.Join(src.Path, "vendor")
	}
	return src.vendor
}

// Load inspects the source code
func (src *GoSource) Load() ([]*packages.Package, error) {
	if src.pkgs == nil {
		cfg := &packages.Config{
			Mode:  packages.LoadImports,
			Error: func(error) {},
		}
		pkgs, err := packages.Load(cfg, filepath.Join(src.Path, "..."))
		if err != nil {
			return nil, errors.Wrap(err, "from Load()")
		}
		src.pkgs = pkgs
	}
	return src.pkgs, nil
}
