// Command resumefetcher is the Resume Fetcher CLI. See CONTEXT.md and
// docs/adr/ for the domain vocabulary and the decisions behind it.
package main

import (
	"fmt"
	"os"

	"resumefetcher/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: resumefetcher <fetch> [flags]")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "fetch":
		err = cli.RunFetch(os.Args[2:], os.Stdin, os.Stdout)
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
