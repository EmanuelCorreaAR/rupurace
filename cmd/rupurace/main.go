// Command rupurace is the RupuRace CLI.
//
// Early development: the engine lives under internal/. Public commands
// (e.g. test) land only when they exist and are tested.
package main

import (
	"fmt"
	"os"

	"github.com/EmanuelCorreaAR/rupurace/internal/version"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-V") {
		fmt.Printf("rupurace %s\n", version.Version)
		return
	}

	fmt.Fprintf(os.Stderr, "rupurace %s — early development; no commands yet\n", version.Version)
	os.Exit(2)
}
