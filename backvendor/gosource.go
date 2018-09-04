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

package backvendor // import "github.com/release-engineering/backvendor/backvendor"

import (
	"path/filepath"
)

// GoSource represents a filesystem tree containing Go source code.
type GoSource string

// Topdir returns the top-level path of the filesystem tree.
func (src GoSource) Topdir() string {
	return string(src)
}

// Vendor returns the path to the vendored source code.
func (src GoSource) Vendor() string {
	return filepath.Join(src.Topdir(), "vendor")
}
