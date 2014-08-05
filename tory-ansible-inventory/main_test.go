package main

import (
	"testing"
)

func TestBuildApp(t *testing.T) {
	app := buildApp()
	if app.Name != "tory-ansible-inventory" {
		t.Fatalf("wrong app name")
	}
}
