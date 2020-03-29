package viewer

import (
	"github.com/pkg/errors"
	"os"
	"path"
	"text/template"
)

type Info struct {
	Id          int
	Title       string
	Author      string
	Description string
	Entries     []Entry
}

type Entry struct {
	Number            int
	Title             string
	Path              string
	ThumbnailFileName string
	ContentFileNames []string
}

var tmpl = template.Must(template.New("viewer").Funcs(template.FuncMap{
	"quoteStr": func(s string) string {
		return "\"" + template.JSEscapeString(s) + "\""
	},
}).ParseFiles("viewer/meta.gojs"))

type Renderer struct {
	BasePath string
}

func (r Renderer) WriteMeta(info Info) error {
	filename := path.Join(r.BasePath, "meta.js")
	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to create file \"%s\"", filename)
	}
	err = tmpl.ExecuteTemplate(file, "meta.gojs", info)
	if err != nil {
		return errors.Wrapf(err, "failed to render template of \"%s\"", filename)
	}
	return nil
}
