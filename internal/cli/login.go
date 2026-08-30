package cli

import (
	"fmt"
	"io"
	"strings"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/platform"
	"resumefetcher/internal/platform/naukriclient"
	"resumefetcher/internal/session"
)

// sessionDir is where captured PlatformClient login sessions persist across
// invocations (ADR-0003: only the session, never credentials).
const sessionDir = "sessions"

// RunLogin implements `login <naukri|linkedin>`: it captures (or reuses) a
// session for the named platform and reports the outcome.
func RunLogin(args []string, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: resumefetcher login <naukri|linkedin>")
	}

	client, source, err := buildLoginClient(strings.ToLower(args[0]), out)
	if err != nil {
		return err
	}
	return runLogin(client, source, out)
}

func runLogin(client platform.Client, source domain.Source, out io.Writer) error {
	if _, err := client.Login(); err != nil {
		return fmt.Errorf("%s login: %w", source, err)
	}
	fmt.Fprintf(out, "%s session is ready.\n", source)
	return nil
}

// buildLoginClient resolves a platform name to its PlatformClient. LinkedIn
// is a distinct, expected "not yet" case (ticket 06) from a genuinely
// invalid platform name.
func buildLoginClient(platformName string, out io.Writer) (platform.Client, domain.Source, error) {
	switch platformName {
	case "naukri":
		return naukriclient.New(session.New(sessionDir), out), domain.SourceNaukri, nil
	case "linkedin":
		return nil, "", fmt.Errorf("linkedin login is not implemented yet (ticket 06)")
	default:
		return nil, "", fmt.Errorf("invalid platform %q: must be naukri or linkedin", platformName)
	}
}
