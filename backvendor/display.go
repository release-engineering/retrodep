package backvendor

import (
	"fmt"
	"io"
	"text/template"
)

const defaultTemplate string = "@{{if .Rev}}{{.Rev}}{{end}}{{if .Tag}} ={{.Tag}}{{end}}{{if .Ver}} ~{{.Ver}}{{end}}"

//TemplateError Returned when there's an error creating a template from the arguments
type TemplateError template.Template

func (t TemplateError) Error(template template.Template) string {
	return fmt.Sprintf("display: Error creating template from %v", template)
}

//Display a Reference using a template
func Display(writer io.Writer, customTemplate string, ref *Reference) error {
	if customTemplate == "" {
		tmpl, err := template.New("output").Parse(defaultTemplate)
		if err != nil {
			return err
		}
		err = tmpl.Execute(writer, ref)
		if err != nil {
			return err
		}
	} else {
		tmpl, err := template.New("output").Parse(customTemplate)
		if err != nil {
			return err
		}
		err = tmpl.Execute(writer, ref)
		if err != nil {
			return err
		}
	}
	return nil
}
