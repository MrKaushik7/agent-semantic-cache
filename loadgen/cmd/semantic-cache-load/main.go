package main

import (
	"fmt"
	"io"
	"os"
)

func run(o io.Writer) {
	fmt.Fprint(o, "Semantic Cache Go Test")
}

func main() {
	run(os.Stdout)
}
