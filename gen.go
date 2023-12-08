//go:build ignore

package main

import (
	_ "embed"
	"os"
	"strings"
	"text/template"
	"time"
)

type typeDef struct {
	Name, Output string
}

var (
	//go:embed error.tmpl
	errorTemplate string
	tmpl          = template.Must(template.New("").Parse(errorTemplate))
	types         = []string{
		"Info",
		"Warn",
		"Error",
		"Fatal",
	}
)

func main() {
	file, err := os.Create("error_gen.go")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	l := len(types)
	data := struct {
		Timestamp string
		Types     []typeDef
	}{
		Timestamp: time.Now().Format(time.RFC3339),
		Types:     make([]typeDef, l),
	}
	for i := 0; i < l; i++ {
		data.Types[i] = typeDef{
			Name:   types[i],
			Output: strings.ToUpper(types[i]),
		}
	}

	tmpl.Execute(file, data)
}
