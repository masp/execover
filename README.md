# About
| :exclamation:  This package requires Go 1.20 or higher  |
|---------------------------------------------------------|

Testing your `main` package has always been challenging with Go, but no longer.

With `Go 1.20` there has been recent improvements that make running your executables with the standard `testing` framework possible.

This package `maintest` makes it even easier to test your executables end-to-end
while still getting all the nice features of built-in Golang testing framework.


# Usage
> See example/ directory for full source code of the below example

Below is a simple executable called `add` that adds two numbers: `add 2 3 -> 5`

`main.go`
```go
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

func main() {
	flag.Parse()
	a, _ := strconv.Atoi(flag.Arg(0))
	b, _ := strconv.Atoi(flag.Arg(1))
	fmt.Printf("%d\n", a+b)
	os.Exit(0)
}
```

With `maintest` we can easily test this by just running our executable directly and asserting on the output:

`main_test.go`
```go
package main

import (
	"os"
	"testing"

	"github.com/masp/maintest"
)

var exe *maintest.Exe

func TestMain(m *testing.M) {
	// Build this package as a test executable called add
	exe, err = maintest.Build("add")
	if err != nil {
		log.Fatal(err)
	}
	
	rc := m.Run()

	// Cleanup and print code coverage
	if err := exe.Finish(); err != nil {
		log.Fatal(err)
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
```

`maintest` will compile the current package once at the start, and then makes it invocable with `exe.Command`. Every run will automatically capture coverage and store it at `coverprofile` and `gocoverdir`.

```
> go test -coverprofile=coverage.txt
> go tool cover -func=coverage.txt                                       
github.com/masp/maintest/example/main.go:10:    main            100.0%
total:                                          (statements)    100.0%
```

| :exclamation:  `go test` will report the coverage as 0% on output, but all other tools will report the correct coverage from `maintest` (including `go tool cover`). This is because `go test` does its own calculations, which can't be overriden.  |
|---------------------------------------------------------|

## Debugging Failing Tests
If a test fails, trying to debug the test like normal will only show the test fixture or what runs the main executable -- not your code that's failing!

To debug the actual executable, you can use the `maintest.DebugFlag`.

```go
var debugFlag = maintest.DebugFlag("example.debug")

func TestMain(m *testing.M) {
	flag.Parse() // flag.Parse must be called before Build
	exe, _ = maintest.Build("add", *debugFlag)
	...
}

func TestAdd4(t *testing.T) {
	cmd, _ := exe.Command("1", "3")
	out, _ := cmd.CombinedOutput() // cmd will block on start and wait for the you to attach
	...
}
```

Run the test

```
go test -test.run TestAdd4 -example.debug 25514
```

and the debugger will listen on `localhost:25514`. You can connect with `dlv` with `dlv connect localhost:25514`, set breakpoints on `main.go` and continue like a normal debugging session. 

If you are using VS code, you can connect by adding this to your `launch.json` configuration:

```
{
	"name": "Attach Integration Test",
	"type": "go",
	"request": "attach",
	"debugAdapter": "dlv-dap",
	"mode": "remote",
	"port": 25514
}
```