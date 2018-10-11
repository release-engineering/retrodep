package backvendor

import (
	"fmt"
	"strings"
	"testing"
)

func TestDisplayDefault(t *testing.T) {
	const expected = "@4309345093405934509 =v0.0.1 ~v0.0.0.20181785"
	ref := &Reference{Rev: "4309345093405934509", Tag: "v0.0.1", Ver: "v0.0.0.20181785"}
	var builder strings.Builder
	var tmpl, err = Display("")
	if err != nil {
		t.Fatal(err)
	}
	err = tmpl.Execute(&builder, ref)
	if err != nil {
		t.Fatal(err)
	}
	if builder.String() != expected {
		t.Fatal(fmt.Sprintf("Expected: %s but got: %s", expected, builder.String()))
	}
}

func TestDisplayGarbage(t *testing.T) {
	var _, err = Display("{{.")
	if err == nil {
		t.Fatal("Should have failed with Error")
	}
}

func TestDisplayNoTag(t *testing.T) {
	const expected = "@4309345093405934509 ~v0.0.0.20181785"
	ref := &Reference{Rev: "4309345093405934509", Ver: "v0.0.0.20181785"}
	var builder strings.Builder
	tmpl, err := Display("")
	if err != nil {
		t.Fatal(err)
	}
	err = tmpl.Execute(&builder, ref)
	if err != nil {
		t.Fatal(err)
	}
	if builder.String() != expected {
		t.Fatal(fmt.Sprintf("Expected: %s but got: %s", expected, builder.String()))
	}
}

func TestDisplayTemplate(t *testing.T) {
	const expected = ":v0.0.0.20181785"
	ref := &Reference{Rev: "4309345093405934509", Ver: "v0.0.0.20181785"}
	var builder strings.Builder
	tmpl, err := Display("{{if .Tag}}:{{.Tag}}{{end}}{{if .Ver}}:{{.Ver}}{{end}}")
	if err != nil {
		t.Fatal(err)
	}
	err = tmpl.Execute(&builder, ref)
	if err != nil {
		t.Fatal(err)
	}
	if builder.String() != expected {
		t.Fatal(fmt.Sprintf("Expected: %s but got: %s", expected, builder.String()))
	}
}

func TestDisplayTemplateElseIf(t *testing.T) {
	const expected = ":v0.0.1"
	ref := &Reference{Rev: "4309345093405934509", Tag: "v0.0.1", Ver: "v0.0.0.20181785"}
	var builder strings.Builder
	tmpl, err := Display("{{if .Tag}}:{{.Tag}}{{else if .Rev}}:{{.Rev}}{{end}}")
	if err != nil {
		t.Fatal(err)
	}
	err = tmpl.Execute(&builder, ref)
	if err != nil {
		t.Fatal(err)
	}
	if builder.String() != expected {
		t.Fatal(fmt.Sprintf("Expected: %s but got: %s", expected, builder.String()))
	}
}

func TestDisplayRepo(t *testing.T) {
	const repo = "https://github.com/release-engineering/backvendor"
	ref := &Reference{Repo: repo}
	var builder strings.Builder
	tmpl, err := Display("{{.Repo}}")
	if err != nil {
		t.Fatal(err)
	}
	err = tmpl.Execute(&builder, ref)
	if err != nil {
		t.Fatal(err)
	}
	if builder.String() != repo {
		t.Errorf("got %q, want %q", builder.String(), repo)
	}
}
