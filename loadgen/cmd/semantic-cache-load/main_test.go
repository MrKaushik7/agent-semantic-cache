package main

import (
	"bytes"
	"testing"
)

func TestRun(t *testing.T) {
	var b bytes.Buffer
	run(&b)

	got := b.String()
	want := "Semantic Cache Go Test"

	if got != want {
		t.Fatalf("run() output = %q, want %q", got, want)
	}
}
