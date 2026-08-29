package reviewer

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"resumefetcher/internal/domain"
)

// Stdin prints extracted Filters and lets the operator accept them as-is or
// edit each field, per the terminal confirm/edit flow.
type Stdin struct {
	in  *bufio.Reader
	out io.Writer
}

func NewStdin(in io.Reader, out io.Writer) *Stdin {
	return &Stdin{in: bufio.NewReader(in), out: out}
}

func (s *Stdin) Review(f domain.Filters) (domain.Filters, error) {
	fmt.Fprintln(s.out, "Extracted search filters:")
	fmt.Fprintf(s.out, "  Skills:        %s\n", strings.Join(f.Skills, ", "))
	fmt.Fprintf(s.out, "  Experience:    %d-%d years\n", f.MinExperience, f.MaxExperience)
	fmt.Fprintf(s.out, "  Location:      %s\n", f.Location)
	fmt.Fprintf(s.out, "  Notice period: %s\n", f.NoticePeriod)
	fmt.Fprint(s.out, "Accept these filters? [Y/n]: ")

	line, err := s.readLine()
	if err != nil {
		return domain.Filters{}, err
	}
	if strings.EqualFold(strings.TrimSpace(line), "n") {
		return s.edit(f)
	}
	return f, nil
}

func (s *Stdin) edit(f domain.Filters) (domain.Filters, error) {
	if skills, err := s.promptString("Skills (comma-separated)", strings.Join(f.Skills, ", ")); err != nil {
		return domain.Filters{}, err
	} else if skills != "" {
		f.Skills = splitAndTrim(skills)
	}

	if v, err := s.promptInt("Min experience (years)", f.MinExperience); err != nil {
		return domain.Filters{}, err
	} else {
		f.MinExperience = v
	}

	if v, err := s.promptInt("Max experience (years)", f.MaxExperience); err != nil {
		return domain.Filters{}, err
	} else {
		f.MaxExperience = v
	}

	if v, err := s.promptString("Location", f.Location); err != nil {
		return domain.Filters{}, err
	} else if v != "" {
		f.Location = v
	}

	if v, err := s.promptString("Notice period", f.NoticePeriod); err != nil {
		return domain.Filters{}, err
	} else if v != "" {
		f.NoticePeriod = v
	}

	return f, nil
}

func (s *Stdin) promptString(label, current string) (string, error) {
	fmt.Fprintf(s.out, "%s [%s]: ", label, current)
	line, err := s.readLine()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (s *Stdin) promptInt(label string, current int) (int, error) {
	raw, err := s.promptString(label, strconv.Itoa(current))
	if err != nil {
		return 0, err
	}
	if raw == "" {
		return current, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		fmt.Fprintf(s.out, "  %q is not a number, keeping %d\n", raw, current)
		return current, nil
	}
	return v, nil
}

func (s *Stdin) readLine() (string, error) {
	line, err := s.in.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return line, nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
