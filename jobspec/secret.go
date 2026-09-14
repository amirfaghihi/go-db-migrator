package jobspec

import (
	"encoding/json"
	"fmt"
)

// Redacted is what a Secret renders as, everywhere, always.
const Redacted = "***"

// Secret is a string that refuses to print itself.
//
// Every path by which a value normally reaches a log line is closed: fmt (all
// verbs, via Formatter), encoding/json, encoding/yaml, and encoding.TextMarshaler.
// Reading the real value requires calling Reveal, which is greppable — so
// "did we leak a password" becomes a question you answer by searching for one
// identifier rather than by auditing every log statement.
type Secret string

// Reveal returns the underlying value. Call sites are deliberately conspicuous.
func (s Secret) Reveal() string { return string(s) }

// Empty reports whether the secret carries no value.
func (s Secret) Empty() bool { return s == "" }

// Format implements fmt.Formatter, which takes precedence over Stringer and
// covers every verb including %#v — the one that a plain String method misses.
func (s Secret) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v', 's', 'q', 'x', 'X':
		_, _ = f.Write([]byte(Redacted))
	default:
		// Unknown verb: still never emit the value.
		_, _ = fmt.Fprintf(f, "%%!%c(jobspec.Secret=%s)", verb, Redacted)
	}
}

func (s Secret) String() string { return Redacted }

func (s Secret) MarshalText() ([]byte, error) { return []byte(Redacted), nil }

func (s Secret) MarshalJSON() ([]byte, error) { return json.Marshal(Redacted) }

func (s Secret) MarshalYAML() (any, error) { return Redacted, nil }

// UnmarshalText, UnmarshalJSON and UnmarshalYAML are deliberately absent.
// Secrets enter the process only through a resolver (see secrets.Resolve), never
// by being decoded straight out of a config file.

var (
	_ fmt.Formatter  = Secret("")
	_ fmt.Stringer   = Secret("")
	_ json.Marshaler = Secret("")
)
