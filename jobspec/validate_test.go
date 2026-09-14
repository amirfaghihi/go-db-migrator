package jobspec

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckNoLiteralSecret(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		// The shape of the DSN this repo shipped in its first commit. The real
		// password is deliberately not reproduced here — see git history.
		{"the shape this repo once shipped", "mongodb://root:REDACTED@10.0.0.1:27017/?authSource=admin", true},
		{"postgres url password", "postgres://app:hunter2@db:5432/orders", true},
		{"mysql url password", "mysql://root:toor@127.0.0.1:3306/app", true},
		{"libpq keyword form", "host=db user=app password=hunter2 dbname=orders", true},
		{"pwd alias", "host=db pwd=hunter2", true},
		{"api key", "https://api.example.com?api_key=abc123", true},

		{"env reference", "${env:SRC_DSN}", false},
		{"file reference", "${file:/run/secrets/dsn}", false},
		{"reference inside a url", "postgres://app:${env:PG_PASSWORD}@db:5432/orders", false},
		{"user but no password", "postgres://app@db:5432/orders", false},
		{"no credentials at all", "postgres://db:5432/orders", false},
		{"sqlite path", "sqlite:///var/lib/app.db", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckNoLiteralSecret("source.dsn", tt.raw)
			if tt.wantErr && err == nil {
				t.Errorf("CheckNoLiteralSecret(%q) = nil, want an error", tt.raw)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("CheckNoLiteralSecret(%q) = %v, want nil", tt.raw, err)
			}
			var lit *ErrLiteralSecret
			if tt.wantErr && err != nil && !errors.As(err, &lit) {
				t.Errorf("error should be *ErrLiteralSecret, got %T", err)
			}
		})
	}
}

// The error tells the operator what to do instead. A rejection that does not
// show the fix just gets overridden with the escape-hatch flag.
func TestLiteralSecretErrorIsActionable(t *testing.T) {
	err := CheckNoLiteralSecret("source.dsn", "postgres://app:hunter2@db/orders")
	if err == nil {
		t.Fatal("want an error")
	}
	for _, want := range []string{"source.dsn", "${env:", "${file:", "--allow-inline-secrets"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error message should mention %q, got:\n%s", want, err)
		}
	}
}
