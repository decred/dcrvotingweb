// Copyright (c) 2017-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"html/template"
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

// BlockLockedIn returns the height of the first block of this agenda's lock-in period. -1 if this agenda has not been locked-in.
func (a *Agenda) BlockLockedIn() int64 {
	if a.IsLockedIn() || a.IsActive() {
		return a.EndHeight + 1
	}
	return -1
}

// BlockActivated returns the height of the first block with this agenda active. -1 if this agenda has not been activated.
func (a *Agenda) BlockActivated() int64 {
	if a.IsActive() {
		return a.BlockLockedIn() + int64(activeNetParams.RuleChangeActivationInterval)
	}
	return -1
}

// DescriptionWithDCPURL writes a new description with an link to any DCP that
// is detected in the text.  It is written to a template.HTML type so the link
// is not escaped when the template is executed.
func (a *Agenda) DescriptionWithDCPURL() template.HTML {
	subst := `<a href="https://github.com/decred/dcps/blob/master/dcp-${1}/dcp-${1}.mediawiki" target="_blank" rel="noopener noreferrer">${0}</a>`
	return template.HTML(dcpRE.ReplaceAllString(a.Description, subst))
}
