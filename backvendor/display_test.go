package backvendor

import (
	"fmt"
	"strings"
	"testing"
)

func TestDisplayDefault(t *testing.T) {
	const expected string = "@4309345093405934509 =v0.0.1 ~v0.0.0.20181785"
	ref := &Reference{Rev: "4309345093405934509", Tag: "v0.0.1", Ver: "v0.0.0.20181785"}
	var builder strings.Builder
	var tmpl, err = Display("")
	if err != nil {
		t.Fatal(err)
	}
	tmpl.Execute(&builder, ref)
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
	const expected string = "@v0.0.0.20181785 ~4309345093405934509"
	ref := &Reference{Rev: "v0.0.0.20181785", Ver: "4309345093405934509"}
	var builder strings.Builder
	var tmpl, err = Display("")
	if err != nil {
		t.Fatal(err)
	}
	tmpl.Execute(&builder, ref)
	if builder.String() != expected {
		t.Fatal(fmt.Sprintf("Expected: %s but got: %s", expected, builder.String()))
	}
}

func TestDisplayTemplate(t *testing.T) {
	const expected string = ":v0.0.0.20181785"
	ref := &Reference{Rev: "v0.0.0.20181785", Ver: "4309345093405934509"}
	var builder strings.Builder
	var tmpl, err = Display("{{if .Tag}}:{{.Tag}}{{end}}{{if .Rev}}:{{.Rev}}{{end}}")
	if err != nil {
		t.Fatal(err)
	}
	tmpl.Execute(&builder, ref)
	if builder.String() != expected {
		t.Fatal(fmt.Sprintf("Expected: %s but got: %s", expected, builder.String()))
	}
}

func TestDisplayTemplateElseIf(t *testing.T) {
	const expected string = ":v0.0.1"
	ref := &Reference{Rev: "4309345093405934509", Tag: "v0.0.1", Ver: "v0.0.0.20181785"}
	var builder strings.Builder
	var tmpl, err = Display("{{if .Tag}}:{{.Tag}}{{else if .Rev}}:{{.Rev}}{{end}}")
	if err != nil {
		t.Fatal(err)
	}
	tmpl.Execute(&builder, ref)
	if builder.String() != expected {
		t.Fatal(fmt.Sprintf("Expected: %s but got: %s", expected, builder.String()))
	}
}
