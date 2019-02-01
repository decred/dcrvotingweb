package agendadb

import (
	"os"
	"testing"

	"github.com/asdine/storm"
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
		ID:     "fakeagenda",
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
	err = db.One("ID", "fakeagenda", &Aloaded)
	if err != nil {
		t.Errorf("Failed to load AgendaTagged: %v", err)
	}

	if Aloaded.Status != A.Status {
		t.Errorf(`Loaded Status incorrect: expected "%s", got "%s"`,
			A.Status, Aloaded.Status)
	}

	if Aloaded.ID != A.ID {
		t.Errorf(`Loaded ID incorrect: expected "%s", got "%s"`,
			A.ID, Aloaded.ID)
	}

	t.Log(Aloaded)
}
