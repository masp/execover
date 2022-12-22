# About
| :exclamation:  This package requires Go 1.20 or higher  |
|---------------------------------------------------------|

With `Go 1.20` there has been recent improvements to code coverage that now allow executables to be instrumented
with better code coverage, along with tooling to make it easy to manage code coverage results. See [Go release notes](https://go.dev/testing/coverage/) for more details.

This package `execover` takes advantage of these changes to make it easy to test your executables end-to-end
while still getting all the nice features of on-demand testing and code coverage.


# Usage
> See example/ directory for full source code of the below example

Given a simple executable called `add` that adds two numbers: `add 2 3 -> 5`

`add.go`
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

With `execover` we can easily test this by running our executable directly:

`add_test.go`
```go
package main

import (
	"os"
	"testing"

	"github.com/masp/execover"
)

var exe *execover.Exe

func TestMain(m *testing.M) {
	exe, _ = execover.Build("add")
	m.Run()
	exe.Finish()
}

func TestAdd4(t *testing.T) {
	out, _ := exe.Command("1", "3").CombinedOutput()
	result := string(out[0])
	if result != "4" {
		t.Errorf("got %s, expected 4", result)
	}
}
```

and now run the above tests:

```
> go1.20rc1 test -coverprofile=coverage.txt
> go tool cover -func=coverage.txt                                       
github.com/masp/execover/example/main.go:10:    main            100.0%
total:                                          (statements)    100.0%
```
