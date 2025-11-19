//go:build tools
// +build tools

// Package tools tracks dependencies for tools used in the build process.
package tools

import (
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)

