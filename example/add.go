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
