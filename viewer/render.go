package viewer

import (
	"github.com/pkg/errors"
	"os"
	"path"
)

type Meta struct {
	BasePath string
	FileName string // defaults to "meta.js"
}

func (m Meta) getFileName() string {
	if m.FileName == "" {
		return "meta.js"
	}
	return m.FileName
}

func (m Meta) Write(info Info) error {
	filename := path.Join(m.BasePath, m.getFileName())
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

type Files struct {
	BasePath string
	FileName string // defaults to "files.js"
}

func (f Files) getFileName() string {
	if f.FileName == "" {
		return "files.js"
	}
	return f.FileName
}

func (f Files) Write(files []string) error {
	filename := path.Join(f.BasePath, f.getFileName())
	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to create file \"%s\"", filename)
	}
	err = tmpl.ExecuteTemplate(file, "files.gojs", files)
	if err != nil {
		return errors.Wrapf(err, "failed to render template of \"%s\"", filename)
	}
	return nil
}
