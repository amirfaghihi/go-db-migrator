package jobspec

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// keywordSecret matches inline credentials in keyword-style DSNs, e.g. the
// libpq form `host=db user=app password=hunter2`.
var keywordSecret = regexp.MustCompile(`(?i)\b(password|passwd|pwd|secret|token|api[_-]?key)\s*=\s*\S+`)

// ErrLiteralSecret reports a credential written directly into a job spec.
type ErrLiteralSecret struct {
	Field string // spec path, e.g. "source.dsn"
	Why   string
}

func (e *ErrLiteralSecret) Error() string {
	return fmt.Sprintf("%s contains a literal credential (%s).\n"+
		"Job specs are meant to be committed, so credentials must be references:\n"+
		"    dsn: ${env:SRC_DSN}\n"+
		"    dsn: ${file:/run/secrets/src_dsn}\n"+
		"Pass --allow-inline-secrets to override.", e.Field, e.Why)
}

// CheckNoLiteralSecret rejects a credential written directly into a spec field.
//
// This runs on the RAW text, before interpolation — afterwards a resolved
// password and a literal one are indistinguishable, which is precisely why the
// check has to happen here.
//
// This rule is a hard error rather than a warning on purpose. The repository
// this tool replaces shipped a live production password in its only commit, and
// a warning would not have stopped it.
func CheckNoLiteralSecret(field, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	// A reference is the correct form; nothing to check.
	if HasReference(raw) {
		return nil
	}

	if u, err := url.Parse(raw); err == nil && u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			return &ErrLiteralSecret{Field: field, Why: "password embedded in the URL"}
		}
	}

	if m := keywordSecret.FindStringSubmatch(raw); m != nil {
		return &ErrLiteralSecret{
			Field: field,
			Why:   fmt.Sprintf("inline %s= parameter", strings.ToLower(m[1])),
		}
	}

	return nil
}
