package main

import (
	"fmt"
	"github.com/hostodo/odo-cli/v2/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
