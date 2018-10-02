package backvendor

import (
	"text/template"
)

const defaultTemplate string = "{{if .Rev}}@{{.Rev}}{{end}}{{if .Tag}} ={{.Tag}}{{end}}{{if .Ver}} ~{{.Ver}}{{end}}"

//Display a Reference using a template
func Display(customTemplate string) (*template.Template, error) {
	if customTemplate == "" {
		customTemplate = defaultTemplate
	}
	return template.New("output").Parse(customTemplate)
}
