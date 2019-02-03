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
	"regexp"
)

// Agenda contains all of the data representing an agenda for the html
// template programming.
type Agenda struct {
	ID                      string
	Status                  string
	Description             string
	QuorumVotedPercentage   float64
	ChoiceIDsActing         []string
	ChoicePercentagesActing []float64
	StartHeight             int64
	EndHeight               int64
	VoteCountPercentage     float64
	BlockLockedIn           int64
	BlockActivated          int64
	BlockForked             int64
}

var dcpRE = regexp.MustCompile(`(?i)DCP\-?(\d{4})`)

// Agenda status may be: started, defined, lockedin, failed, active

// IsActive indicates if the agenda is active
func (a *Agenda) IsActive() bool {
	return a.Status == "active"
}

// IsStarted indicates if the agenda is started
func (a *Agenda) IsStarted() bool {
	return a.Status == "started"
}

// IsDefined indicates if the agenda is defined
func (a *Agenda) IsDefined() bool {
	return a.Status == "defined"
}

// IsLockedIn indicates if the agenda is lockedin
func (a *Agenda) IsLockedIn() bool {
	return a.Status == "lockedin"
}

// IsFailed indicates if the agenda is failed
func (a *Agenda) IsFailed() bool {
	return a.Status == "failed"
}

// DescriptionWithDCPURL writes a new description with an link to any DCP that
// is detected in the text.  It is written to a template.HTML type so the link
// is not escaped when the template is executed.
func (a *Agenda) DescriptionWithDCPURL() template.HTML {
	subst := `<a href="https://github.com/decred/dcps/blob/master/dcp-${1}/dcp-${1}.mediawiki" target="_blank" rel="noopener noreferrer">${0}</a>`
	return template.HTML(dcpRE.ReplaceAllString(a.Description, subst))
}

var funcMap = template.FuncMap{
	"plus":      plus,
	"minus":     minus,
	"minus64":   minus64,
	"modiszero": modiszero,
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
func modiszero(a, b int) bool {
	return (a % b) == 0
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

// pickNoun returns the singular or plural form of a noun depending
// on the count n.
func pickNoun(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// ceilDiv returns the ceiling of the result of dividing the given value.
func ceilDiv(numerator, denominator int) int {
	return int(math.Ceil(float64(numerator) / float64(denominator)))
}

// blocksToTimeEstimate returns a human-readable estimate for the amount of time
// a given number of blocks would take.
func blocksToTimeEstimate(blocksRemaining int) string {
	remainingSecs := blocksRemaining * int(activeNetParams.TargetTimePerBlock.Seconds())
	if remainingSecs > hourCutoffSecs {
		value := ceilDiv(remainingSecs, secondsPerDay)
		noun := pickNoun(value, "day", "days")
		return fmt.Sprintf("%d %s", value, noun)
	} else if remainingSecs > minuteCutoffSecs {
		value := ceilDiv(remainingSecs, secondsPerHour)
		noun := pickNoun(value, "hour", "hours")
		return fmt.Sprintf("%d %s", value, noun)
	}

	value := ceilDiv(remainingSecs, secondsPerMinute)
	noun := pickNoun(value, "minute", "minutes")
	return fmt.Sprintf("%d %s", value, noun)
}
