package backvendor

import (
	"strings"
	"testing"
)

func TestDisplay(t *testing.T) {
	ref := &Reference{Rev: "4309345093405934509", Tag: "v0.0.1", Ver: "v0.0.0.20181785"}
	var builder strings.Builder
	err := Display(&builder, "", ref)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDisplayGarbage(t *testing.T) {
	ref := &Reference{Rev: "4309345093405934509", Tag: "v0.0.1", Ver: "v0.0.0.20181785"}
	var builder strings.Builder
	err := Display(&builder, "{{.", ref)
	if err == nil {
		t.Fatal("Should have failed with Error")
	}
}

func TestDisplayNoTag(t *testing.T) {
	ref := &Reference{Rev: "4309345093405934509", Ver: "v0.0.0.20181785"}
	var builder strings.Builder
	err := Display(&builder, "", ref)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDisplayTemplate(t *testing.T) {
	ref := &Reference{Rev: "4309345093405934509", Ver: "v0.0.0.20181785"}
	var builder strings.Builder
	err := Display(&builder, "{{if .Tag}}:{{.Tag}}{{end}}{{if .Ver}}:{{.Ver}}{{end}}", ref)
	if err != nil {
		t.Fatal(err)
	}

}
