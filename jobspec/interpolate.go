package jobspec

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// refPattern matches a single ${scheme:argument} reference.
var refPattern = regexp.MustCompile(`\$\{([a-zA-Z][a-zA-Z0-9_-]*):([^}]*)\}`)

// Resolver turns a reference argument into its value.
type Resolver interface {
	// Scheme is the prefix this resolver answers to, e.g. "env".
	Scheme() string
	// Resolve returns the value for one reference argument.
	Resolve(arg string) (string, error)
}

// EnvResolver reads ${env:NAME} from the process environment.
type EnvResolver struct{}

func (EnvResolver) Scheme() string { return "env" }

func (EnvResolver) Resolve(arg string) (string, error) {
	v, ok := os.LookupEnv(arg)
	if !ok {
		return "", fmt.Errorf("environment variable %s is not set", arg)
	}
	if v == "" {
		return "", fmt.Errorf("environment variable %s is set but empty", arg)
	}
	return v, nil
}

// FileResolver reads ${file:/path} from disk, trimming trailing whitespace —
// the newline that every `echo secret > file` leaves behind breaks DSNs in ways
// that are tedious to debug.
type FileResolver struct{}

func (FileResolver) Scheme() string { return "file" }

func (FileResolver) Resolve(arg string) (string, error) {
	b, err := os.ReadFile(arg)
	if err != nil {
		return "", fmt.Errorf("reading secret file %s: %w", arg, err)
	}
	v := strings.TrimRight(string(b), " \t\r\n")
	if v == "" {
		return "", fmt.Errorf("secret file %s is empty", arg)
	}
	return v, nil
}

// DefaultResolvers are the resolvers available without extra configuration.
// Cloud secret managers and the OS keyring are added in a later phase; each is
// just another Resolver in this slice.
func DefaultResolvers() []Resolver {
	return []Resolver{EnvResolver{}, FileResolver{}}
}

// Interpolate replaces every ${scheme:arg} reference in s.
//
// The returned error never contains a resolved value, only the reference that
// failed — an interpolation error message is one of the easier places to leak a
// secret into a terminal that someone then pastes into a bug report.
func Interpolate(s string, resolvers []Resolver) (string, error) {
	byScheme := make(map[string]Resolver, len(resolvers))
	for _, r := range resolvers {
		byScheme[r.Scheme()] = r
	}

	var firstErr error
	out := refPattern.ReplaceAllStringFunc(s, func(match string) string {
		if firstErr != nil {
			return match
		}
		m := refPattern.FindStringSubmatch(match)
		scheme, arg := m[1], m[2]

		r, ok := byScheme[scheme]
		if !ok {
			firstErr = fmt.Errorf("unknown reference scheme %q in ${%s:...}; known schemes: %s",
				scheme, scheme, strings.Join(schemeNames(resolvers), ", "))
			return match
		}
		v, err := r.Resolve(arg)
		if err != nil {
			firstErr = fmt.Errorf("resolving ${%s:%s}: %w", scheme, arg, err)
			return match
		}
		return v
	})

	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}

// HasReference reports whether s contains any ${scheme:arg} reference.
func HasReference(s string) bool { return refPattern.MatchString(s) }

func schemeNames(rs []Resolver) []string {
	n := make([]string, 0, len(rs))
	for _, r := range rs {
		n = append(n, r.Scheme())
	}
	return n
}
