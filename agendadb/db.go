package agendadb

import (
	"github.com/decred/dcrd/dcrjson"
)

// AgendaTagged has the same fields as dcrjson.Agenda, but with the Id field
// marked as the primary key via the `storm:"id"` tag. Fields tagged for
// indexing by the DB are: StartTime, ExpireTime, Status, and QuorumProgress.
type AgendaTagged struct {
	Id             string           `json:"id" storm:"id"`
	Description    string           `json:"description"`
	Mask           uint16           `json:"mask"`
	StartTime      uint64           `json:"starttime" storm:"index"`
	ExpireTime     uint64           `json:"expiretime" storm:"index"`
	Status         string           `json:"status" storm:"index"`
	QuorumProgress float64          `json:"quorumprogress" storm:"index"`
	Choices        []dcrjson.Choice `json:"choices"`
}

// ChoiceLabeled embeds dcrjson.Choice along with the AgendaID for the choice,
// and a string array suitable for use as a primary key. The AgendaID is tagged
// as an index for quick lookups based on the agenda.
type ChoiceLabeled struct {
	AgendaChoice   [2]string `storm:"id"`
	AgendaID       string    `json:"agendaid" storm:"index"`
	dcrjson.Choice `storm:"inline"`
}
