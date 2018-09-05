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

import "golang.org/x/tools/go/vcs"

// vcsGit describes how to use Git. (From golang.org/x/tools/go/vcs/vcs.go)
var vcsGit = &vcs.Cmd{
	Name:        "Git",
	Cmd:         "git",
	CreateCmd:   "clone {repo} {dir}",
	DownloadCmd: "pull --ff-only",
	TagCmd: []vcs.TagCmd{
		{"show-ref", `(?:tags|origin)/(\S+)$`},
	},
	TagLookupCmd: []vcs.TagCmd{
		{"show-ref tags/{tag} origin/{tag}", `((?:tags|origin)/\S+)$`},
	},
	TagSyncCmd:     "checkout {tag}",
	TagSyncDefault: "checkout master",

	Scheme:  []string{"git", "https", "http", "git+ssh"},
	PingCmd: "ls-remote {scheme}://{repo}",
}
