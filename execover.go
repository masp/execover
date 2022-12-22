// Package execover makes it easy to create test executables for integration tests with excellent support for code coverage.
//
// execover must be used within the `main` package within the `TestMain` function.
package execover

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Exe is an instrumented executable that can be easily run in integration tests.
//
// To execute, use Command(args...).
//
// After all commands have finished, Finish must be called for the coverage data to be written.
type Exe struct {
	Path        string // Path is the full path to the executable file useful for passing to exec.Command
	CoverageDir string // Directory where coverage results will temporarily go after every invocation

	binDir string // temporary dir used to store coverage info, built executable, etc...
}

// Build builds the named executable using the local module (meant to be run in TestMain). The test executable
// will be called exeName and placed in a temporary directory. The executable can be run
func Build(exeName string) (*Exe, error) {
	gotool, err := goTool()
	if err != nil {
		return nil, err
	}
	binDir, err := os.MkdirTemp("", "gotestexe")
	if err != nil {
		return nil, err
	}

	covDir := filepath.Join(binDir, ".coverage")
	if err := os.Mkdir(covDir, 0766); err != nil {
		return nil, err
	}
	if runtime.GOOS == "window" {
		exeName += ".exe"
	}
	testExe := filepath.Join(binDir, exeName)
	build := exec.Command(gotool, "build", "-cover", "-o", testExe)
	if err != nil {
		return nil, err
	}

	out, err := build.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go build: %s", string(out))
	}
	return &Exe{Path: testExe, CoverageDir: covDir, binDir: binDir}, nil
}

// Command returns Cmd ready to be run that will invoke the built test executable and output the results
// to the shared coverage directory.
func (b *Exe) Command(args ...string) *exec.Cmd {
	cmd := exec.Command(b.Path, args...)
	cmd.Env = append(os.Environ(), "GOCOVERDIR="+b.CoverageDir)
	return cmd
}

// Finish will merge all the coverage from the previous executions and write the output to -coverprofile.
// This func uses the `go tool covdata` command with textfmt for backwards compatibility with existing
// tools, e.g. `go tool covdata textfmt -i b.CoverageDir -o '-coverprofile'`. -coverprofile is parsed from os.Args.
func (b *Exe) Finish() error {
	defer os.RemoveAll(b.binDir)

	gotool, err := goTool()
	if err != nil {
		return err
	}

	var covdata *exec.Cmd
	coverprofile := findArg("test.coverprofile")
	if coverprofile != "" {
		covdata = exec.Command(gotool, "tool", "covdata", "textfmt", "-i", b.CoverageDir, "-o", coverprofile)
	} else {
		return fmt.Errorf("-test.coverprofile and -test.gocoverdir not set in os.Args, not writing coverage data")
	}

	out, err := covdata.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go tool covdata: %s", string(out))
	}
	return nil
}

func findArg(key string) string {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-"+key) {
			vs := strings.Split(arg, "=")
			if len(vs) <= 1 {
				return ""
			}
			return strings.Join(vs[1:], "=")
		}
	}
	return ""
}

// GoTool reports the path to the Go tool.
func goTool() (string, error) {
	var exeSuffix string
	if runtime.GOOS == "windows" {
		exeSuffix = ".exe"
	}
	goBin, err := exec.LookPath("go1.20rc1" + exeSuffix)
	if err != nil {
		return "", errors.New("cannot find go tool: " + err.Error())
	}
	return goBin, nil
}
