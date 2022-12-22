package main

import (
	"log"
	"os"
	"testing"

	"github.com/masp/execover"
)

var exe *execover.Exe

func TestMain(m *testing.M) {
	var err error
	exe, err = execover.Build("add")
	if err != nil {
		log.Fatalf("build error: %v", err)
	}

	rc := m.Run()
	if err := exe.Finish(); err != nil {
		log.Printf("warning: %v", err)
	}
	os.Exit(rc)
}

func TestAdd4(t *testing.T) {
	out, _ := exe.Command("1", "3").CombinedOutput()
	result := string(out[0])
	if result != "4" {
		t.Errorf("got %s, expected 4", result)
	}
}
