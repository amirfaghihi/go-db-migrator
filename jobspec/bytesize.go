package jobspec

import (
	"fmt"
	"strconv"
	"strings"
)

// ByteSize is a byte count that accepts human units in YAML: 4MiB, 512KB, 2GiB.
type ByteSize int64

var byteUnits = []struct {
	suffix string
	mult   int64
}{
	{"KIB", 1 << 10}, {"MIB", 1 << 20}, {"GIB", 1 << 30}, {"TIB", 1 << 40},
	{"KB", 1000}, {"MB", 1000 * 1000}, {"GB", 1000 * 1000 * 1000}, {"TB", 1e12},
	{"K", 1 << 10}, {"M", 1 << 20}, {"G", 1 << 30}, {"T", 1 << 40},
	{"B", 1},
}

func (b *ByteSize) UnmarshalYAML(unmarshal func(any) error) error {
	// A bare number is a byte count.
	var n int64
	if err := unmarshal(&n); err == nil {
		*b = ByteSize(n)
		return nil
	}

	var s string
	if err := unmarshal(&s); err != nil {
		return fmt.Errorf("byte size must be a number or a string like \"4MiB\"")
	}
	v, err := ParseByteSize(s)
	if err != nil {
		return err
	}
	*b = v
	return nil
}

// ParseByteSize parses "4MiB", "512KB", "1024" into a byte count.
func ParseByteSize(s string) (ByteSize, error) {
	t := strings.ToUpper(strings.TrimSpace(s))
	if t == "" {
		return 0, fmt.Errorf("empty byte size")
	}
	for _, u := range byteUnits {
		if !strings.HasSuffix(t, u.suffix) {
			continue
		}
		num := strings.TrimSpace(strings.TrimSuffix(t, u.suffix))
		f, err := strconv.ParseFloat(num, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid byte size %q", s)
		}
		if f < 0 {
			return 0, fmt.Errorf("byte size %q is negative", s)
		}
		return ByteSize(f * float64(u.mult)), nil
	}
	n, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid byte size %q (want a number or a value like \"4MiB\")", s)
	}
	return ByteSize(n), nil
}

func (b ByteSize) String() string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1fGiB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1fMiB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fKiB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%dB", int64(b))
	}
}
