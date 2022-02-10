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

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// TODO: shouldn't need to send every template the time built.
	// Render this once
	renderTemplate(w, "home.page.tmpl", PageData{Time: app.buildTime})
}

func articleHandler(w http.ResponseWriter, r *http.Request) {
	relURL := r.URL.Path[len("/articles/"):]
	if relURL == "" {
		renderTemplate(w, "articles.page.tmpl", PageData{Time: app.buildTime})
	} else if relURL == "go-talking-to-java" {
		renderTemplate(w, "go-talking-to-java.page.tmpl", PageData{Time: app.buildTime})
	} else {
		http.NotFound(w, r)
	}
}

func ademHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := PageData{Time: app.buildTime, QueryInput: "6 4 2 + 2 10"}
		renderTemplate(w, "adem.page.tmpl", data)
	}
	// POST is called when the user submits a query
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		var data PageData

		// Validate form input as a string consisting of
		// Only numbers, spaces, or +
		re := regexp.MustCompile("^[0-9 \\+]*$")
		query := r.PostFormValue("query")

		if re.MatchString(query) {
			data, err = runAdemWithQuery(query)
			if err != nil {
				data = PageData{Time: app.buildTime, QueryResult: []string{"An error occurred in the backend while computing the Adem relations."}}
			}
		} else {
			data = PageData{Time: app.buildTime, QueryResult: []string{"Wrong query form."}}
		}

		renderTemplate(w, "adem.page.tmpl", data)
	}
}

// Run adem.py with a query
// The query should already have been validated as proper input, but if not, the python program itself will (hopefully) fail
func runAdemWithQuery(query string) (PageData, error) {
	pythonString := fmt.Sprintf(`import adem; adem.print_adem("%s")`, query)

	cmd := exec.Command("python3", "-c", pythonString)
	cmd.Dir = "./bin/adem" // should be in another location?

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return PageData{}, err
	}

	// TODO: if the command is taking too long, say >30s, then kill it
	// This can kill server resources
	start := time.Now()

	err = cmd.Start()
	if err != nil {
		return PageData{}, err
	}

	b, err := io.ReadAll(stdout)
	if err != nil {
		return PageData{}, err
	}

	// An error here means that the python command went poorly AKA exit code 1, likely an overflow
	// If a different error occurs, the end user will still think it's an overflow...
	err = cmd.Wait()
	if err != nil {
		return PageData{Time: app.buildTime, QueryInput: "", QueryResult: []string{"Error: overflow?"}}, nil
	}

	t := time.Now()
	elapsed := t.Sub(start)

	output := []string{
		fmt.Sprintf("Result: %s", string(b)),
		fmt.Sprintf("Time elapsed: %s", elapsed),
	}

	return PageData{Time: app.buildTime, QueryInput: query, QueryResult: output}, nil
}
