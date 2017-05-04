package agendadb

import (
	"os"
	"testing"

	"github.com/asdine/storm"
	"github.com/decred/dcrd/dcrjson"
)

func TestAgendaDBOpenClose(t *testing.T) {
	dbName := "testdb.db"
	if _, err := os.Stat(dbName); err == nil {
		t.Log("Deleting existing DB files")
		if err = os.Remove(dbName); err != nil {
			t.Fatalf("Failed to delete existing DB: %v", err)
		}
	}

	// Open a new database
	adb, err := Open(dbName)
	if err != nil {
		t.Fatalf("Failed to open new DB: %v", err)
	}

	// Initialize buckets
	if err = adb.sdb.Init(&AgendaTagged{}); err != nil {
		t.Errorf("Failed to Init AgendaTagged bucket: %v", err)
	}
	if err = adb.sdb.Init(&ChoiceLabeled{}); err != nil {
		t.Errorf("Failed to Init ChoiceLabeled bucket: %v", err)
	}

	if err = adb.Close(); err != nil {
		t.Fatalf("Failed to close DB: %v", err)
	}

	// Open and existing database
	adb, err = Open(dbName)
	if err != nil {
		t.Fatalf("Failed to open existing DB: %v", err)
	}

	if err = adb.Close(); err != nil {
		t.Fatalf("Failed to close DB: %v", err)
	}

	os.Remove(dbName)
}

func TestSaveLoadAgenda(t *testing.T) {
	A := AgendaTagged{
		Id:     "fakeagenda",
		Status: "active",
	}

	db, err := storm.Open("testdb.db")
	if err != nil {
		t.Errorf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err = db.Init(&AgendaTagged{}); err != nil {
		t.Errorf("Failed to init DB: %v", err)
	}

	err = db.Save(&A)
	if err != nil {
		t.Errorf("Failed to save AgendaTagged: %v", err)
	}

	var Aloaded AgendaTagged
	err = db.One("Id", "fakeagenda", &Aloaded)
	if err != nil {
		t.Errorf("Failed to load AgendaTagged: %v", err)
	}

	if Aloaded.Status != A.Status {
		t.Errorf(`Loaded Status incorrect: expected "%s", got "%s"`,
			A.Status, Aloaded.Status)
	}

	if Aloaded.Id != A.Id {
		t.Errorf(`Loaded Id incorrect: expected "%s", got "%s"`,
			A.Id, Aloaded.Id)
	}

	t.Log(Aloaded)
}

func TestSaveLoadChoice(t *testing.T) {
	pk0 := [2]string{"stuff", "abstain"}
	C0 := ChoiceLabeled{
		AgendaChoice: pk0,
		AgendaID:     pk0[0],
		Choice: dcrjson.Choice{
			Id:          pk0[1],
			Description: "abstain voting on stuff",
			Bits:        0,
			IsAbstain:   true,
			IsNo:        false,
			Count:       345,
		},
	}

	db, err := storm.Open("testdb.db")
	if err != nil {
		t.Errorf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err = db.Init(&ChoiceLabeled{}); err != nil {
		t.Errorf("Failed to init DB: %v", err)
	}

	err = db.Save(&C0)
	if err != nil {
		t.Errorf("Failed to save ChoiceLabeled: %v", err)
	}

	var C0loaded ChoiceLabeled
	err = db.One("AgendaChoice", pk0, &C0loaded)
	if err != nil {
		t.Errorf("Failed to load AgendaTagged: %v", err)
	}

	if C0loaded.Description != C0.Description {
		t.Errorf(`Loaded Description incorrect: expected "%s", got "%s"`,
			C0.Description, C0loaded.Description)
	}

	if C0loaded.Id != C0.Id {
		t.Errorf(`Loaded Id incorrect: expected "%s", got "%s"`,
			C0.Id, C0loaded.Id)
	}

	t.Log(C0loaded)

	// Save another choice
	pk1 := [2]string{"stuff", "yes"}
	C1 := ChoiceLabeled{
		AgendaChoice: pk1,
		AgendaID:     pk1[0],
		Choice: dcrjson.Choice{
			Id:          pk1[1],
			Description: "yay for stuff",
			Bits:        4,
			IsAbstain:   false,
			IsNo:        false,
			Count:       321,
		},
	}

	err = db.Save(&C1)
	if err != nil {
		t.Errorf("Failed to save ChoiceLabeled: %v", err)
	}

	var C1loaded ChoiceLabeled
	err = db.One("AgendaChoice", pk1, &C1loaded)
	if err != nil {
		t.Errorf("Failed to load ChoiceLabeled: %v", err)
	}

	if C1loaded.Description != C1.Description {
		t.Errorf(`Loaded Description incorrect: expected "%s", got "%s"`,
			C0.Description, C0loaded.Description)
	}

	if C1loaded.Id != C1.Id {
		t.Errorf(`Loaded Id incorrect: expected "%s", got "%s"`,
			C1.Id, C1loaded.Id)
	}

	t.Log(C1loaded)

	// Test updating a field of a stored Choice. Give a choice some more votes.
	extraVotes := uint32(47)
	err = db.UpdateField(&ChoiceLabeled{AgendaChoice: pk1},
		"Count", C1.Count+extraVotes)
	if err != nil {
		t.Error("Failed to update field:", err)
	}

	C1loaded = ChoiceLabeled{}
	if err = db.One("AgendaChoice", pk1, &C1loaded); err != nil {
		t.Error("Failed to load AgendaChoice:", err)
	}

	if C1loaded.Count != C1.Count+extraVotes {
		t.Errorf("Loaded Count incorrect. Expected %v, got %v",
			C1.Count+extraVotes, C1loaded.Count)
	}

	// must reindex after updating a field
	db.ReIndex(&ChoiceLabeled{})

	t.Log(C1loaded)

	// Another choice on a different agenda from the two above
	pk2 := [2]string{"things", "yes"}
	C2 := ChoiceLabeled{
		AgendaChoice: pk2,
		AgendaID:     pk2[0],
		Choice: dcrjson.Choice{
			Id:          pk2[1],
			Description: "yay for things",
			Bits:        4,
			IsAbstain:   false,
			IsNo:        false,
			Count:       444,
		},
	}

	err = db.Save(&C2)
	if err != nil {
		t.Errorf("Failed to save ChoiceLabeled: %v", err)
	}

	// Get both choices for the first agenda
	var choicesLoaded []ChoiceLabeled
	err = db.Find("AgendaID", "stuff", &choicesLoaded)
	if err != nil {
		t.Errorf("Failed to load AgendaTagged: %v", err)
	}

	if len(choicesLoaded) != 2 {
		t.Errorf("Error loading choices. Expected %v, got %v", 2, len(choicesLoaded))
	}

	t.Log(choicesLoaded)

	// Get the choice for the second agenda
	choicesLoaded = nil
	err = db.Find("AgendaID", "things", &choicesLoaded)
	if err != nil {
		t.Errorf("Failed to load AgendaTagged: %v", err)
	}

	if len(choicesLoaded) != 1 {
		t.Errorf("Error loading choices. Expected %v, got %v", 1, len(choicesLoaded))
	}

	t.Log(choicesLoaded)
}
