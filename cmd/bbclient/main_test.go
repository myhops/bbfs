package main

import "testing"

func TestTags(t *testing.T) {
	args := []string{"bbclient", "-project-key", "~zandp06"}

	getenv := func(_ string) string {return ""}

	if err := run(args, getenv); err != nil {
		t.Fatalf("error: %s", err.Error())
	}
}
