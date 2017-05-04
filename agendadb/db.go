package agendadb

import (
	"fmt"
	"os"

	"github.com/asdine/storm"
	"github.com/decred/dcrd/dcrjson"
)

type AgendaDB struct {
	sdb        *storm.DB
	NumAgendas int
	NumChoices int
}

func Open(dbPath string) (*AgendaDB, error) {
	_, err := os.Stat(dbPath)
	isNewDB := os.IsNotExist(err)

	db, err := storm.Open(dbPath)
	if err != nil {
		return nil, err
	}

	var numAgendas, numChoices int
	if !isNewDB {
		numAgendas, err = db.Count(&AgendaTagged{})
		if err != nil {
			fmt.Printf("Unable to count agendas in existing DB: %v\n", err)
		}
		numChoices, err = db.Count(&ChoiceLabeled{})
		if err != nil {
			fmt.Printf("Unable to count choices in existing DB: %v\n", err)
		}
		fmt.Printf("Opened existing datatbase with %d agendas.\n", numAgendas)
	}

	agendaDB := &AgendaDB{
		sdb:        db,
		NumAgendas: numAgendas,
		NumChoices: numChoices,
	}
	return agendaDB, err
}

func (db *AgendaDB) Close() error {
	return db.sdb.Close()
}

func (db *AgendaDB) StoreAgenda(agenda *AgendaTagged) error {
	if db == nil || db.sdb == nil {
		return fmt.Errorf("AgendaDB not initialized")
	}
	return db.sdb.Save(agenda)
}

func (db *AgendaDB) LoadAgenda(agendaID string) (*AgendaTagged, error) {
	if db == nil || db.sdb == nil {
		return nil, fmt.Errorf("AgendaDB not initialized")
	}
	agenda := new(AgendaTagged)
	if err := db.sdb.One("Id", agendaID, agenda); err != nil {
		return nil, err
	}
	return agenda, nil
}

func (db *AgendaDB) ListAgendas() error {
	if db == nil || db.sdb == nil {
		return fmt.Errorf("AgendaDB not initialized")
	}
	q := db.sdb.Select().OrderBy("StartTime")
	i := 0
	return q.Each(new(AgendaTagged), func(record interface{}) error {
		a := record.(*AgendaTagged)
		fmt.Printf("%d: %s\n", i, a.Id)
		i++
		return nil
	})
}

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
