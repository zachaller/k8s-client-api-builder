package main

import (
	"fmt"
	"os"

	"github.com/example/my-platform/cmd/my-platform/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
