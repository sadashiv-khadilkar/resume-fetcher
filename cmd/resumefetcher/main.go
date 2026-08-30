// Command resumefetcher is the Resume Fetcher CLI. See CONTEXT.md and
// docs/adr/ for the domain vocabulary and the decisions behind it.
package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"resumefetcher/internal/cli"
)

func main() {
	// .env is optional (e.g. LLM_PROVIDER unset defaults to the fake
	// provider, which needs no key) and gitignored - see .env.example.
	// Values already in the environment take precedence over the file.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "warning: failed to load .env:", err)
	}

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
