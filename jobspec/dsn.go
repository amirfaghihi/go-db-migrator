package jobspec

import (
	"fmt"
	"net/url"
	"strings"
)

// DSN is a parsed connection string. The credential-bearing form is held as a
// Secret and is reachable only via Reveal; everything else about the endpoint
// (scheme, host, database) is safe to print and is what Display renders.
//
// DSN itself implements fmt.Formatter so that even `log.Printf("%v", dsn)` —
// the mistake everyone makes once — emits the redacted display form.
type DSN struct {
	Scheme   string // "postgres", "mongodb", "mysql", "sqlite"
	Host     string // host[:port], or host list for mongodb+srv
	Database string

	raw Secret // the full connection string, credentials included
}

// ParseDSN parses a connection string. The input is treated as secret from the
// moment it arrives, whatever its contents.
func ParseDSN(s string) (DSN, error) {
	if strings.TrimSpace(s) == "" {
		return DSN{}, fmt.Errorf("empty DSN")
	}
	u, err := url.Parse(s)
	if err != nil {
		// Deliberately not wrapping err: url.Error embeds the input string,
		// which is the credential we are trying not to print.
		return DSN{}, fmt.Errorf("malformed DSN (parse failed)")
	}
	if u.Scheme == "" {
		return DSN{}, fmt.Errorf("DSN has no scheme")
	}
	return DSN{
		Scheme:   u.Scheme,
		Host:     u.Host,
		Database: strings.TrimPrefix(u.Path, "/"),
		raw:      Secret(s),
	}, nil
}

// Reveal returns the full connection string for handing to a driver.
func (d DSN) Reveal() string { return d.raw.Reveal() }

// Display is the safe rendering: scheme, host and database, no credentials,
// no query parameters (which carry passwords often enough to not be worth it).
func (d DSN) Display() string {
	if d.Scheme == "" {
		return "<empty dsn>"
	}
	var b strings.Builder
	b.WriteString(d.Scheme)
	b.WriteString("://")
	if d.Host != "" {
		b.WriteString(d.Host)
	}
	if d.Database != "" {
		b.WriteString("/")
		b.WriteString(d.Database)
	}
	return b.String()
}

func (d DSN) Empty() bool { return d.raw.Empty() }

func (d DSN) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v', 's', 'q':
		_, _ = f.Write([]byte(d.Display()))
	default:
		_, _ = fmt.Fprintf(f, "%%!%c(jobspec.DSN=%s)", verb, d.Display())
	}
}

func (d DSN) String() string { return d.Display() }

func (d DSN) MarshalText() ([]byte, error) { return []byte(d.Display()), nil }

func (d DSN) MarshalYAML() (any, error) { return d.Display(), nil }

var (
	_ fmt.Formatter = DSN{}
	_ fmt.Stringer  = DSN{}
)
