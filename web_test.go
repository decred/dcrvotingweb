package main

import (
	"testing"
)

func TestIsDCP(t *testing.T) {
	strTests := []struct {
		s string
		r bool
	}{
		{"DCP0001", true},
		{"CP0001", false},
		{"dcp0001", true},
		{"DCP-0001", true},
		{"DTP0001", false},
	}

	a := Agenda{}

	for _, ts := range strTests {
		a.Description = ts.s
		if ts.r != a.IsDCP() {
			t.Errorf("%s should have been %v", ts.s, ts.r)
		}
	}
}
