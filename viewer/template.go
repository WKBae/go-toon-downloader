package viewer

import "text/template"

var tmpl = template.Must(template.New("viewer").Funcs(template.FuncMap{
	"quoteStr": func(s string) string {
		return "\"" + template.JSEscapeString(s) + "\""
	},
}).ParseFiles("viewer/meta.gojs", "viewer/files.gojs"))
