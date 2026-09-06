package main

import (
	"fmt"
	"os"
)

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "grillme:", err)
	os.Exit(1)
}
