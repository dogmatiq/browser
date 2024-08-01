package web

import (
	"embed"
	"html/template"
	"io/fs"
	"path"
	"strings"

	"github.com/gin-gonic/contrib/renders/multitemplate"
)

var (
	//go:embed *.html
	//go:embed */*.html
	//go:embed */*/*.html
	templateFS    embed.FS
	pageTemplates multitemplate.Render
)

func init() {
	funcs := template.FuncMap{}

	// Add a function for rendering each of the component templates.
	entries, err := templateFS.ReadDir("components")
	if err != nil {
		panic(err)
	}

	for _, e := range entries {
		funcName := strings.TrimSuffix(e.Name(), ".html")
		fileName := path.Join("components", e.Name())

		tmpl := template.Must(
			template.ParseFS(
				templateFS,
				fileName,
			),
		)

		funcs[funcName] = func(data interface{}) (template.HTML, error) {
			w := &strings.Builder{}

			if err := tmpl.Execute(w, data); err != nil {
				return "", err
			}

			return template.HTML(w.String()), nil
		}
	}

	// Add page templates to the renderer used by the Gin framework.
	pageTemplates = multitemplate.New()

	if err := fs.WalkDir(
		templateFS,
		"pages",
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if entry.IsDir() {
				return nil
			}

			tmpl := template.Must(
				template.
					New("").
					Funcs(funcs).
					ParseFS(templateFS, "template.html", path),
			)

			pageTemplates.Add(
				strings.TrimPrefix(path, "pages/"),
				tmpl.Lookup("template.html"),
			)

			return nil
		},
	); err != nil {
		panic(err)
	}
}
