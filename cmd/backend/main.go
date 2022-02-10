package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var port = os.Getenv("PORT")
var dirStatic = "./static"
var dirTemplate = "./templates"
var app *application // TODO: shouldn't be global

type application struct {
	buildTime     string
	templateCache map[string]*template.Template
}

type PageData struct {
	QueryInput  string
	QueryResult []string
	Time        string
}

func main() {
	// bitwise or to include both date and time
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	// shortfile includes file and line number where error occurred
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	if !isInteger(port) {
		errorLog.Fatal("Invalid or missing port!")
	}
	infoLog.Printf("Server started on port %s", port)

	templateCache, err := parseTemplates()
	if err != nil {
		errorLog.Fatal(err)
	}

	buildTime := time.Now().UTC().Format("Jan 2, 2006 15:04:05 UTC")

	app = &application{
		buildTime:     buildTime,
		templateCache: templateCache,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/articles/", articleHandler)
	mux.HandleFunc("/adem/", ademHandler)
	mux.HandleFunc("/", homeHandler)

	// Relative to project root dir
	// TODO: do we really want this to be browse-able?
	fileServer := http.FileServer(http.Dir("./static/"))

	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	err = http.ListenAndServe(":"+port, mux)
	errorLog.Fatal(err)
}

// Parse all templates and return a cache of them
func parseTemplates() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	// Get a slice of all the *.page.tmpl files
	// These are `top-level` templates which will eventually be rendered to the user
	pages, err := filepath.Glob(filepath.Join(dirTemplate, "*.page.tmpl"))
	if err != nil {
		return nil, err
	}

	// Need to loop through each page because the layout and partial templates need to
	//   be rendered separately for each.
	// See: https://pkg.go.dev/text/template#hdr-Nested_template_definitions
	for _, page := range pages {
		// name will be of the form "blah.page.tmpl"
		name := filepath.Base(page)

		ts, err := template.ParseFiles(page)
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseGlob(filepath.Join(dirTemplate, "*.layout.tmpl"))
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseGlob(filepath.Join(dirTemplate, "*.partial.tmpl"))
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}

func renderTemplate(w http.ResponseWriter, name string, data PageData) {
	ts, ok := app.templateCache[name]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	err := ts.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
