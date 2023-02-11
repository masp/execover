package main

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/masp/maintest/example/add"
)

func main() {
	flag.Parse()
	a, _ := strconv.Atoi(flag.Arg(0))
	b, _ := strconv.Atoi(flag.Arg(1))
	fmt.Printf("%d\n", add.Add(a, b))
}
