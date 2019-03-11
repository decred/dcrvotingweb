package main

import (
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
)

var funcMap = template.FuncMap{
	"plus":                 plus,
	"minus":                minus,
	"minus64":              minus64,
	"commaSeparate":        commaSeparate,
	"twoDecimalPlaces":     twoDecimalPlaces,
	"blocksToTimeEstimate": blocksToTimeEstimate,
}

func plus(a, b int) int {
	return a + b
}
func minus(a, b int) int {
	return a - b
}
func minus64(a, b int64) int64 {
	return a - b
}
func commaSeparate(number int64) string {
	return humanize.Comma(number)
}
func twoDecimalPlaces(number float64) string {
	number = math.Floor(number*100) / 100
	return fmt.Sprintf("%.2f", number)
}

// renders the 'home' template which is currently located at "start.html".
func (td *WebUI) homePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	err := td.templ.Execute(w, td.TemplateData)
	if err != nil {
		log.Printf("Failed to Execute: %v", err)
		return
	}
	// TODO: Use TemplateExecToString only when the template data is updated
	// (i.e. block notification).
}

// WebUI represents the html web interface. It includes the template related
// data, methods for parsing the templates, and the http.HandlerFuncs registered
// with URL paths by the http router.
type WebUI struct {
	TemplateData *templateFields
	templ        *template.Template
	templFiles   []string
}

// NewWebUI is the constructor for WebUI.  It creates a html/template.Template,
// loads the function map, and parses the template files.
func NewWebUI() (*WebUI, error) {
	fp := filepath.Join("public", "views", "start.html")
	tmpl, err := template.New("home").Funcs(funcMap).ParseFiles(fp)
	if err != nil {
		return nil, err
	}

	// may have multiple template files eventually
	templFiles := []string{fp}

	return &WebUI{
		templ:      tmpl,
		templFiles: templFiles,
	}, nil
}

// ParseTemplates parses the html templates into a new html/template.Temlate.
func (td *WebUI) ParseTemplates() (err error) {
	td.templ, err = template.New("home").ParseFiles(td.templFiles...)
	return
}

// See reloadsig*.go for an exported method
func (td *WebUI) reloadTemplatesSig(sig os.Signal) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, sig)

	go func() {
		for {
			sigr := <-sigChan
			log.Printf("Received %s", sig)
			if sigr == sig {
				if err := td.ParseTemplates(); err != nil {
					log.Println(err)
					continue
				}
				log.Println("Web UI html templates reparsed.")
			}
		}
	}()
}

const (
	secondsPerMinute = 60
	secondsPerHour   = 60 * secondsPerMinute
	secondsPerDay    = 24 * secondsPerHour
	hourCutoffSecs   = 72 * secondsPerHour
	minuteCutoffSecs = 2 * secondsPerHour
)

// ceilDiv returns the ceiling of the result of dividing the given value.
func ceilDiv(numerator, denominator int64) int {
	return int(math.Ceil(float64(numerator) / float64(denominator)))
}

// blocksToTimeEstimate returns a human-readable estimate for the amount of time
// a given number of blocks would take.
func blocksToTimeEstimate(startHeight int64, currentHeight int64) string {
	blocksRemaining := startHeight - currentHeight
	remainingSecs := blocksRemaining * int64(activeNetParams.TargetTimePerBlock.Seconds())
	if remainingSecs > hourCutoffSecs {
		value := ceilDiv(remainingSecs, secondsPerDay)
		return english.Plural(value, "day", "")
	} else if remainingSecs > minuteCutoffSecs {
		value := ceilDiv(remainingSecs, secondsPerHour)
		return english.Plural(value, "hour", "")
	}

	value := ceilDiv(remainingSecs, secondsPerMinute)
	return english.Plural(value, "minute", "")
}
