package main

import (
	"flag"
	"log"
	"os"
	"testing"

	"github.com/masp/maintest"
)

var exe *maintest.Exe
var debugFlag = maintest.DebugFlag("add.debug")

func TestMain(m *testing.M) {
	maintest.DebugLog = log.New(os.Stderr, "[maintest] ", 0)
	flag.Parse()

	var err error
	exe, err = maintest.Build("add", *debugFlag)
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
	out, _ := exe.Command("1", "3").Output()
	result := string(out[0])
	if result != "4" {
		t.Errorf("got %s, expected 4", string(out))
	}
}
