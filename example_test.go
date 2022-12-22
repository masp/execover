package execover

import (
	"log"
	"testing"
)

var exe *Exe

func TestMain(m *testing.M) {
	var err error
	exe, err = Build("execover")
	if err != nil {
		log.Fatalf("build: %v", err)
	}

	m.Run()
	if err := exe.Finish(); err != nil {
		log.Printf("warning: %v", err)
	}
}
