package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"
)

var port = os.Getenv("PORT")
var webRoot = os.Getenv("WEBROOT") // location of html files

func main() {
	if port == "" {
		log.Fatal("No port was specified!")
	}

	fs := http.FileServer(http.Dir(webRoot))
	http.Handle("/", fs)
	http.HandleFunc("/adem/", ademHandler)
	http.HandleFunc("/adem/query", ademQueryHandler)
	//http.HandleFunc("/request/", printRequest)

	http.ListenAndServe(":"+port, nil)
}

// Print out the request...
func printRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Type: %s\n", r.Method)
	fmt.Fprintf(w, "Protocol: %s\n", r.Proto)
	fmt.Fprintf(w, "Header: %v\n", r.Header)
	fmt.Fprintf(w, "Body: %v\n", r.Body)
}

func ademHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "adem.html")
}

func ademQueryHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// validate form input as a string consisting of
	// only numbers, spaces, +
	re := regexp.MustCompile("^[0-9 \\+]*$")
	query := r.PostFormValue("query")
	if !re.MatchString(query) {
		fmt.Fprintf(w, "Wrong query form.")
		return
	}

	fmt.Fprintf(w, "Query: %s\n", query)

	pythonString := fmt.Sprintf(`import adem; adem.print_adem("%s")`, query)

	cmd := exec.Command("python3", "-c", pythonString)
	cmd.Dir = "bin/adem" // should be in another location?

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now()

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	b, err := io.ReadAll(stdout)
	if err != nil {
		fmt.Fprintf(w, "Command ran with error: %v", err)
	}

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

	t := time.Now()
	elapsed := t.Sub(start)

	fmt.Fprintf(w, "Output: %s", string(b))
	fmt.Fprintf(w, "Time elapsed: %s", elapsed)
}
