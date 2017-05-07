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
		{" CP0001", false},
		{"XCP0001", false},
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

func TestDCPNumber(t *testing.T) {
	strTests := []struct {
		s string
		r string
	}{
		{"DCP0001", "0001"},
		{"CP0001", ""},
		{"dcp0001", "0001"},
		{"DCP-0001", "0001"},
		{"DTP0001", ""},
	}

	a := Agenda{}

	for _, ts := range strTests {
		a.Description = ts.s
		if ts.r != a.DCPNumber() {
			t.Errorf("%s should have been %v", ts.s, ts.r)
		}
	}
}
