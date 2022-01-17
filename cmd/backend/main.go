package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	if port == "" {
		log.Fatal("No port was specified!")
	}

	templateCache, err := parseTemplates()
	if err != nil {
		log.Fatal(err)
	}

	buildTime := time.Now().UTC().Format("Jan 2, 2006 15:04:05 UTC")

	app = &application{
		buildTime:     buildTime,
		templateCache: templateCache,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/adem/", ademHandler)
	mux.HandleFunc("/", homeHandler)

	// Relative to project root dir
	fileServer := http.FileServer(http.Dir("./static/"))

	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	err = http.ListenAndServe(":"+port, mux)
	log.Fatal(err)
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

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// TODO: shouldn't need to send every template the time built.
	// Render this once
	renderTemplate(w, "home.page.tmpl", PageData{Time: app.buildTime})
}

func ademHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := PageData{Time: app.buildTime, QueryInput: "6 4 2 + 2 10"}
		renderTemplate(w, "adem.page.tmpl", data)
	}
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		var data PageData

		// validate form input as a string consisting of
		// only numbers, spaces, +
		re := regexp.MustCompile("^[0-9 \\+]*$")
		query := r.PostFormValue("query")

		// TODO: really shouldn't put so much in an else block, this should be refactored
		if !re.MatchString(query) {
			data = PageData{Time: app.buildTime, QueryResult: []string{"Wrong query form."}}
		} else {
			pythonString := fmt.Sprintf(`import adem; adem.print_adem("%s")`, query)

			cmd := exec.Command("python3", "-c", pythonString)
			cmd.Dir = "./bin/adem" // should be in another location?

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				log.Fatal(err)
			}

			// TODO: if the command is taking too long, say >30s, then kill it
			// This can kill server resources
			start := time.Now()

			err = cmd.Start()
			if err != nil {
				log.Fatal(err)
			}

			b, err := io.ReadAll(stdout)
			if err != nil {
				fmt.Fprintf(w, "Command ran with error: %v", err)
				return
			}

			// An error here means that the python command went poorly AKA exit code 1, likely overflow
			// TODO: Handle this better...
			//   Better to use Stderr and have a better error handling for the webserver
			err = cmd.Wait()
			if err != nil {
				data = PageData{Time: app.buildTime, QueryInput: "", QueryResult: []string{"Overflow!"}}
			} else {
				t := time.Now()
				elapsed := t.Sub(start)

				output := []string{
					fmt.Sprintf("Result: %s", string(b)),
					fmt.Sprintf("Time elapsed: %s", elapsed),
				}

				// Note that query has passed the regex validation at this point
				data = PageData{Time: app.buildTime, QueryInput: query, QueryResult: output}
			}
		}

		renderTemplate(w, "adem.page.tmpl", data)
	}
}
