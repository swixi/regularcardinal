package main

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"time"
)

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
	} else if relURL == "graphs-in-go" {
		renderTemplate(w, "graphs-in-go.page.tmpl", PageData{Time: app.buildTime})
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
