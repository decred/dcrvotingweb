package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"

	"github.com/decred/dcrd/dcrjson"
)

// Agenda embeds the Agenda returned by getvoteinfo with several fields to
// facilitate the html template programming.
type Agenda struct {
	dcrjson.Agenda            `storm:"inline"`
	QuorumExpirationDate      string
	QuorumVotedPercentage     float64
	QuorumAbstainedPercentage float64
	ChoiceIDs                 []string
	ChoicePercentages         []float64
	ChoiceIDsActing           []string
	ChoicePercentagesActing   []float64
	StartHeight               int64
	VoteCountPercentage       float64
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

// IsDCP indicates if agenda has a DCP paper
func (a *Agenda) IsDCP() bool {
	return dcpRE.MatchString(a.Description)
}

// DCPNumber gets the DCP number as a string with any leading zeros
func (a *Agenda) DCPNumber() string {
	if a.IsDCP() {
		matches := dcpRE.FindStringSubmatch(a.Description)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// DescriptionWithDCPURL writes a new description with an link to any DCP that
// is detected in the text.  It is written to a template.HTML type so the link
// is not escaped when the template is executed.
func (a *Agenda) DescriptionWithDCPURL() template.HTML {
	subst := `<a href="https://github.com/decred/dcps/blob/master/dcp-${1}/dcp-${1}.mediawiki" target="_blank">${0}</a>`
	return template.HTML(dcpRE.ReplaceAllString(a.Description, subst))
}

// Overall data structure given to the template to render.
type templateFields struct {

	// Network
	Network string

	// Basic information
	BlockHeight uint32
	// Link to current block on explorer
	BlockExplorerLink string
	// BlockVersion Information
	//
	// BlockVersions is the data after it has been prepared for graphing.
	BlockVersions map[int32]*blockVersions
	// BlockVersionHeights is an array of Block heights for graph's x axis.
	BlockVersionsHeights []int64
	// BlockVersionSuccess is a bool whether or not BlockVersion has
	// successfully tripped over to the new version.
	BlockVersionSuccess bool
	// BlockVersionWindowLength is the activeNetParams of BlockUpgradeNumToCheck
	// rolling window length.
	BlockVersionWindowLength uint64
	// BlockVersionEnforceThreshold is the activeNetParams of BlockEnforceNumRequired.
	BlockVersionEnforceThreshold int
	// BlockVersionRejectThreshold is the activeNetParams of BlockRejectNumRequired.
	BlockVersionRejectThreshold int
	// BlockVersionCurrent is the currently calculated block version based on the rolling window.
	BlockVersionCurrent int32
	// BlockVersionMostPopular is the calculated most popular block version that is NOT current version.
	BlockVersionMostPopular int32
	// BlockVersionMostPopularPercentage is the percentage of the most popular block version
	BlockVersionMostPopularPercentage float64
	// BlockVersionNext is teh next block version.
	BlockVersionNext int32
	// BlockVersionNextPercentage is the share of the next block version in the current rolling window.
	BlockVersionNextPercentage float64

	// StakeVersion Information
	//
	// StakeVersionThreshold is the activeNetParams of StakeVersion threshold made into a float for display
	StakeVersionThreshold float64
	// StakeVersionWindowLength is the activeNetParams of StakeVersionInterval
	StakeVersionWindowLength int64
	// StakeVersionIntervalBlocks shows the actual blocks for the current window
	StakeVersionIntervalBlocks string
	// StakeVersionWindowVoteTotal is the number of total possible votes in the windows.
	// It is reduced by number of observed missed votes thus far in the window.
	StakeVersionWindowVoteTotal int64
	// StakeVersionIntervalLabels are labels for the bar graph for each of the past 4 fixed stake version intervals.
	StakeVersionIntervalLabels []string
	// StakeVersionVotesRemaining is the calculated number of votes possibly remaining in the current stake version interval.
	StakeVersionVotesRemaining int64
	// StakeVersionsIntervals  is the data received from GetStakeVersionInfo json-rpc call to dcrd.
	StakeVersionsIntervals []dcrjson.VersionInterval
	// StakeVersionIntervalResults is the data after being analyzed for graph displaying.
	StakeVersionIntervalResults []intervalVersionCounts
	// StakeVersionSuccess is a bool for whether or not the StakeVersion has rolled over in this window.
	StakeVersionSuccess bool
	// StakeVersionCurrent is the StakeVersion that has been seen in the recent block header.
	StakeVersionCurrent uint32
	// StakeVersionMostPopular is the most popular stake version that is NOT the current stake version.
	StakeVersionMostPopular uint32
	// StakeVersionMostPopularCount is the count of most popular stake versions.
	StakeVersionMostPopularCount uint32
	// StakeVersionMostPopularPercentage is the percentage of most popular stake versions out of possible votes.
	StakeVersionMostPopularPercentage float64
	// StakeVersionRequiredVotes is the number of stake version votes required for the stake version to change.
	StakeVersionRequiredVotes int32
	// StakeVersionTimeRemaining is a string to show how much estimated time is remaining in the stake version interval.
	StakeVersionTimeRemaining string
	// Quorum and Rule Change Information
	// RuleChangeActivationQuorum is the activeNetParams of RuleChangeActivationQuorum
	RuleChangeActivationQuorum uint32
	// Quorum is a bool that is true if needed number of yes/nos were
	// received (>10%).
	Quorum bool
	// QuorumThreshold is the percentage required for the RuleChange to become active.
	QuorumThreshold float64
	// LockedinPercentage is the percent of the voing window remaining
	LockedinPercentage float64
	// Length of the static rule change interval
	RuleChangeActivationInterval int64
	// Agendas contains all the agendas and their statuses
	Agendas []Agenda
	// Phase Upgrading or Voting
	IsUpgrading bool

	// GetVoteInfoResult has all the raw data returned from getvoteinfo json-rpc command.
	GetVoteInfoResult *dcrjson.GetVoteInfoResult
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

// TemplateExecToString executes the specified template with given data, writing
// the output into a string.
func TemplateExecToString(t *template.Template, name string, data interface{}) (string, error) {
	var page bytes.Buffer
	err := t.ExecuteTemplate(&page, name, data)
	return page.String(), err
}

// renders the 'home' template that is current located at "design_sketch.html".
func (td *WebUI) demoPage(w http.ResponseWriter, r *http.Request) {
	err := td.templ.Execute(w, td.TemplateData)
	if err != nil {
		fmt.Printf("Failed to Execute: %v\n", err)
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
			fmt.Printf("Received %s\n", sig)
			if sigr == sig {
				if err := td.ParseTemplates(); err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Println("Web UI html templates reparsed.")
			}
		}
	}()
}
