package main

import (
	"testing"

	"github.com/asdine/storm"
)

func TestSaveLoadAgenda(t *testing.T) {
	A := Agenda{
		Id:     "fakeagenda",
		Status: "active",
	}

	db, err := storm.Open("testdb.db")
	if err != nil {
		t.Errorf("Failed to open DB")
	}
	defer db.Close()

	err = db.Save(&A)
	if err != nil {
		t.Errorf("Failed to save Agenda")
	}

	var Aloaded Agenda
	err = db.One("Id", "fakeagenda", &Aloaded)
	if err != nil {
		t.Errorf("Failed to load Agenda")
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
