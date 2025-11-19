package main

import (
	"fmt"
	"os"

	"github.com/zachaller/k8s-client-api-builder/cmd/krm-sdk/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
