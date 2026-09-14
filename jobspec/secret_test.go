package jobspec

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const canary = "hunter2-do-not-print"

// TestSecretNeverPrints covers every route by which a value normally reaches a
// log line. Each of these is a real way credentials have escaped in real tools.
func TestSecretNeverPrints(t *testing.T) {
	s := Secret(canary)

	renders := map[string]string{
		"%v":          fmt.Sprintf("%v", s),
		"%s":          fmt.Sprintf("%s", s),
		"%q":          fmt.Sprintf("%q", s),
		"%#v":         fmt.Sprintf("%#v", s),
		"%x":          fmt.Sprintf("%x", s),
		"%d":          fmt.Sprintf("%d", s),
		"Sprint":      fmt.Sprint(s),
		"Sprintln":    fmt.Sprintln(s),
		"String":      s.String(),
		"in a struct": fmt.Sprintf("%v", struct{ DSN Secret }{s}),
		"in a slice":  fmt.Sprintf("%v", []Secret{s}),
		"in a map":    fmt.Sprintf("%v", map[string]Secret{"dsn": s}),
		"wrapped err": fmt.Errorf("connect failed: %w", fmt.Errorf("dsn %v", s)).Error(),
	}

	for name, got := range renders {
		if strings.Contains(got, canary) {
			t.Errorf("%s leaked the secret: %s", name, got)
		}
	}

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(b), canary) {
		t.Errorf("json.Marshal leaked the secret: %s", b)
	}

	y, err := yaml.Marshal(s)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	if strings.Contains(string(y), canary) {
		t.Errorf("yaml.Marshal leaked the secret: %s", y)
	}
}

func TestSecretReveal(t *testing.T) {
	if got := Secret(canary).Reveal(); got != canary {
		t.Errorf("Reveal() = %q, want %q", got, canary)
	}
}

func TestDSNNeverPrints(t *testing.T) {
	dsn, err := ParseDSN("postgres://app:" + canary + "@db.example.com:5432/orders?sslmode=require")
	if err != nil {
		t.Fatalf("ParseDSN: %v", err)
	}

	for _, got := range []string{
		fmt.Sprintf("%v", dsn),
		fmt.Sprintf("%s", dsn),
		fmt.Sprintf("%q", dsn),
		fmt.Sprintf("%#v", dsn),
		fmt.Sprint(dsn),
		dsn.String(),
		dsn.Display(),
		fmt.Sprintf("%v", struct{ D DSN }{dsn}),
		fmt.Errorf("connect: %v", dsn).Error(),
	} {
		if strings.Contains(got, canary) {
			t.Errorf("DSN leaked the password: %s", got)
		}
	}

	y, err := yaml.Marshal(dsn)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	if strings.Contains(string(y), canary) {
		t.Errorf("yaml.Marshal leaked the password: %s", y)
	}

	if want := "postgres://db.example.com:5432/orders"; dsn.Display() != want {
		t.Errorf("Display() = %q, want %q", dsn.Display(), want)
	}
	if !strings.Contains(dsn.Reveal(), canary) {
		t.Error("Reveal() should return the full DSN")
	}
}

// A malformed DSN must not echo its input: url.Error embeds the string it
// failed to parse, which is exactly the credential we are protecting.
func TestParseDSNErrorDoesNotEchoInput(t *testing.T) {
	_, err := ParseDSN("postgres://app:" + canary + "@%%%bad-host/db")
	if err == nil {
		t.Skip("input parsed successfully; nothing to check")
	}
	if strings.Contains(err.Error(), canary) {
		t.Errorf("parse error leaked the input: %v", err)
	}
}
