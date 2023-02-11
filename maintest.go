// Package maintest makes it easy to create end-to-end tests for executables with support
// for code coverage, debugging, and more.
//
// maintest must be used within the `main` package within the `TestMain` function.
package maintest

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var (
	DebugLog *log.Logger = log.New(io.Discard, "", 0)
)

// Exe is an instrumented executable that can be easily run in integration tests.
//
// To execute, use Command(args...).
//
// After all commands have finished, Finish must be called for the coverage data to be written.
type Exe struct {
	Path        string // Path is the full path to the executable file useful for passing to exec.Command
	CoverageDir string // Directory where coverage results will temporarily go after every invocation
	binDir      string // temporary dir used to store coverage info, built executable, etc...
	buildPkg    string // package name to build, e.g. github.com/masp/maintest/example, defaults to .

	extraArgs      []string // extra build args to pass when the executable is being built by go build
	overrideCovDir string   // override of where to place the final merged coverage after executing all the tests (overrides -coverprofile flag)
	delveOpts      []string // if non-empty, will execute exes with dlv exec [delveOpts] [exe...] to allow interactive debugging

}

type Option func(e *Exe)

// DebugFlag will add a flag to the command line that can be specified to enable debug mode
func DebugFlag(name string) *Option {
	var opt Option
	opt = func(e *Exe) {}
	flag.Func(name, "Stop test on start with dlv listening in headless on port given (e.g. -flag.debug=22565)", func(s string) error {
		_, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		opt = Debug("--headless", "-l", "localhost:"+s)
		return nil
	})

	return &opt
}

// Debug builds the executable with debug symbols enabled and optionally starts dlv for debugging.
//
// This is useful if you want to attach and debug your executable in your integration tests while they run.
// Using dlv exec (see https://github.com/go-delve/delve/blob/master/Documentation/usage/dlv_exec.md) allows a test to
// be "paused" and attached to interactively, which makes it possible to debug the actual executable rather than the test
// fixture which is much less interesting. dlvArgs will be passed to the run like `dlv exec [dlvArgs...] [exeAndFlags...]`
func Debug(dlvArgs ...string) Option {
	return func(e *Exe) {
		e.extraArgs = append(e.extraArgs, `-gcflags=all=-N -l`) // make sure debug flags are enabled when using dlv
		e.delveOpts = append(e.delveOpts, dlvArgs...)
	}
}

// WriteCoverage redirects the coverage from -coverprofile to override path.
func WriteCoverage(path string) Option {
	return func(e *Exe) {
		e.overrideCovDir = path
	}
}

// Package will cause `go build` to run on a different package than
// the current directory. The package must have an executable (`package main`).
func Package(pkg string) Option {
	return func(e *Exe) {
		e.buildPkg = pkg
	}
}

// Build builds the named executable using the local module (meant to be run in TestMain). The test executable
// will be called exeName and placed in a temporary directory. The executable can be run
func Build(exeName string, opts ...Option) (*Exe, error) {
	gotool, err := goTool()
	if err != nil {
		return nil, err
	}
	var exe Exe
	for _, option := range opts {
		option(&exe)
	}

	exe.binDir, err = os.MkdirTemp("", "gotestexe")
	if err != nil {
		return nil, err
	}

	exe.CoverageDir = filepath.Join(exe.binDir, ".coverage")
	err = os.MkdirAll(exe.CoverageDir, 0700)
	if err != nil {
		return nil, err
	}

	DebugLog.Printf("GOCOVERDIR: %s", exe.CoverageDir)
	if runtime.GOOS == "window" {
		exeName += ".exe"
	}
	exe.Path = filepath.Join(exe.binDir, exeName)

	args := []string{"build"}
	args = append(args, exe.extraArgs...)
	args = append(args, "-cover", "-o", exe.Path)
	if exe.buildPkg != "" {
		args = append(args, exe.buildPkg)
	}
	build := exec.Command(gotool, args...)
	DebugLog.Printf("build: %s", strings.Join(build.Args, " "))
	if err != nil {
		return nil, err
	}

	out, err := build.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go build: %s", string(out))
	}
	return &exe, nil
}

// Command returns Cmd ready to be run that will invoke the built test executable and output the results
// to the shared coverage directory.
func (b *Exe) Command(args ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if len(b.delveOpts) > 0 {
		dlvArgs := []string{"exec"}
		dlvArgs = append(dlvArgs, "--log-dest", "/dev/null")
		dlvArgs = append(dlvArgs, b.delveOpts...)
		dlvArgs = append(dlvArgs, b.Path, "--")
		dlvArgs = append(dlvArgs, args...)
		dlvExe := "dlv"
		if runtime.GOOS == "windows" {
			dlvExe += ".exe"
		}
		cmd = exec.Command(dlvExe, dlvArgs...)
	} else {
		cmd = exec.Command(b.Path, args...)
	}

	cmd.Env = append(os.Environ(), "GOCOVERDIR="+b.CoverageDir)
	return cmd
}

// Finish will merge all the coverage from the previous executions and write the output to -coverprofile.
// This func uses the `go tool covdata` command with textfmt for backwards compatibility with existing
// tools, e.g. `go tool covdata textfmt -i b.CoverageDir -o '-coverprofile'`. -coverprofile is parsed from os.Args.
func (b *Exe) Finish() error {
	defer os.RemoveAll(b.binDir)
	DebugLog.Printf("go test args: %s", strings.Join(os.Args, " "))

	coverprofile := findArg("test.coverprofile")
	if b.overrideCovDir != "" {
		coverprofile = b.overrideCovDir
	}
	if coverprofile != "" { // merge the output of the executable to the coverprofile dir as well
		err := mergeGoCover(b.CoverageDir, coverprofile)
		if err != nil {
			return err
		}
	}

	gocoverdir := findGoCoverDir()
	if gocoverdir != "" {
		// Copy all the coverage files to the configured directory
		DebugLog.Printf("copying coverage from %s to %s", b.CoverageDir, gocoverdir)
		err := os.MkdirAll(gocoverdir, 0766)
		if err != nil {
			return fmt.Errorf("mkdir GOCOVERDIR: %w", err)
		}
		err = copyAll(b.CoverageDir, gocoverdir)
		if err != nil {
			return fmt.Errorf("copying cov files to -test.gocoverdir %s: %v", gocoverdir, err)
		}
	}

	if coverprofile == "" && gocoverdir == "" {
		return fmt.Errorf("-test.coverprofile and -test.gocoverdir not set in os.Args, not writing coverage data")
	}
	return nil
}

// mergeGoCover takes the new binary coverage files and merges them all to a dst file
func mergeGoCover(from, dst string) error {
	gotool, err := goTool()
	if err != nil {
		return err
	}
	covdata := exec.Command(gotool, "tool", "covdata", "textfmt", "-i", from, "-o", dst)
	DebugLog.Printf("%s", strings.Join(covdata.Args, " "))
	out, err := covdata.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go tool covdata: %s", string(out))
	}
	return nil
}

func findGoCoverDir() string {
	d := findArg("test.gocoverdir")
	if d == "" {
		d = os.Getenv("GOCOVERDIR")
	}
	return d
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

// copyAll copies all the coverage files from src to dst folders
func copyAll(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		DebugLog.Printf("found %s", entry.Name())
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			continue
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	return err
}

// GoTool reports the path to the Go tool.
func goTool() (string, error) {
	var exeSuffix string
	if runtime.GOOS == "windows" {
		exeSuffix = ".exe"
	}
	goBin, err := exec.LookPath("go" + exeSuffix)
	if err != nil {
		return "", errors.New("cannot find go tool: " + err.Error())
	}
	return goBin, nil
}
