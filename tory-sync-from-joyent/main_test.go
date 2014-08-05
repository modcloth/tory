package main

import "testing"

func TestBuildApp(t *testing.T) {
	app := buildApp()
	if app.Name != "tory-sync-from-joyent" {
		t.Fatalf("wrong app name")
	}
}
