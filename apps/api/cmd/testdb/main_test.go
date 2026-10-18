package main

import "testing"

func TestProvisioningBoundary(t *testing.T) {
	valid := "postgres://moetopia:placeholder@192.168.101.41:55432/moetopia_dev"
	name := "moetopia_test_0123456789abcdef"
	for _, test := range []struct {
		dsn, name, action string
		ok                bool
	}{
		{valid, name, "create", true},
		{valid, name, "drop", true},
		{"postgres://moetopia@localhost/production", name, "drop", false},
		{"postgres://postgres@localhost/moetopia_dev", name, "create", false},
		{valid, "moetopia_dev", "drop", false},
		{valid, name + `"; DROP DATABASE postgres;--`, "drop", false},
		{valid, name, "reset", false},
	} {
		if err := validate(test.dsn, test.name, test.action); (err == nil) != test.ok {
			t.Errorf("unexpected validation result for action %q and target %q", test.action, test.name)
		}
	}
}
