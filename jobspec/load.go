package jobspec

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"gopkg.in/yaml.v3"
)

// LoadOptions tune how strictly a spec is read.
type LoadOptions struct {
	// Resolvers handle ${scheme:arg} references. Nil means DefaultResolvers.
	Resolvers []Resolver
	// AllowInlineSecrets downgrades the literal-credential check to nothing.
	// Exposed as --allow-inline-secrets, for people who insist.
	AllowInlineSecrets bool
	// SkipResolve parses and validates without touching the environment or the
	// filesystem, so `migrator validate` works with no credentials present.
	SkipResolve bool
}

// Load reads, validates and resolves a job spec.
//
// The order matters and is not interchangeable: decode strictly, then check for
// literal credentials on the RAW text, then interpolate. Checking after
// interpolation cannot distinguish a resolved secret from a committed one.
func Load(path string, opts LoadOptions) (*Spec, *Resolved, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("opening job spec: %w", err)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	// A typo'd key is a mistake, not a default. yaml.v2 silently ignored these,
	// which is how a spec quietly runs with settings nobody chose.
	dec.KnownFields(true)

	var spec Spec
	if err := dec.Decode(&spec); err != nil {
		return nil, nil, fmt.Errorf("parsing job spec %s: %w", path, err)
	}

	if err := spec.Validate(opts); err != nil {
		return nil, nil, err
	}

	spec.applyDefaults()

	if opts.SkipResolve {
		return &spec, &Resolved{}, nil
	}

	resolvers := opts.Resolvers
	if resolvers == nil {
		resolvers = DefaultResolvers()
	}

	var resolved Resolved
	if resolved.Source, err = resolveDSN("source.dsn", spec.Source.DSN, resolvers); err != nil {
		return nil, nil, err
	}
	if resolved.Sink, err = resolveDSN("sink.dsn", spec.Sink.DSN, resolvers); err != nil {
		return nil, nil, err
	}

	return &spec, &resolved, nil
}

func resolveDSN(field, raw string, resolvers []Resolver) (DSN, error) {
	s, err := Interpolate(raw, resolvers)
	if err != nil {
		return DSN{}, fmt.Errorf("%s: %w", field, err)
	}
	dsn, err := ParseDSN(s)
	if err != nil {
		return DSN{}, fmt.Errorf("%s: %w", field, err)
	}
	return dsn, nil
}

// Validate checks the spec's shape. It runs before interpolation, so it can see
// literal credentials, and it never needs the environment to be populated.
func (s *Spec) Validate(opts LoadOptions) error {
	var errs []error

	if s.Version == 0 {
		errs = append(errs, errors.New("version is required (currently: 1)"))
	} else if s.Version != 1 {
		errs = append(errs, fmt.Errorf("unsupported spec version %d (this binary understands 1)", s.Version))
	}

	if s.Source.DSN == "" {
		errs = append(errs, errors.New("source.dsn is required"))
	}
	if s.Sink.DSN == "" {
		errs = append(errs, errors.New("sink.dsn is required"))
	}

	if !opts.AllowInlineSecrets {
		if err := CheckNoLiteralSecret("source.dsn", s.Source.DSN); err != nil {
			errs = append(errs, err)
		}
		if err := CheckNoLiteralSecret("sink.dsn", s.Sink.DSN); err != nil {
			errs = append(errs, err)
		}
	}

	if len(s.Objects) == 0 {
		errs = append(errs, errors.New("at least one entry under objects is required"))
	}
	for i, o := range s.Objects {
		if o.Source == "" && o.SourcePattern == "" {
			errs = append(errs, fmt.Errorf("objects[%d]: one of source or source_pattern is required", i))
		}
		if o.Source != "" && o.SourcePattern != "" {
			errs = append(errs, fmt.Errorf("objects[%d]: source and source_pattern are mutually exclusive", i))
		}
		if o.SourcePattern != "" && o.Target != "" {
			errs = append(errs, fmt.Errorf("objects[%d]: source_pattern matches many objects, so use target_template, not target", i))
		}
		if err := validateMode(o.Mode); err != nil {
			errs = append(errs, fmt.Errorf("objects[%d]: %w", i, err))
		}
	}

	if err := validateMode(s.Defaults.Mode); err != nil {
		errs = append(errs, fmt.Errorf("defaults: %w", err))
	}
	if s.Defaults.ReadPartitions < 0 {
		errs = append(errs, errors.New("defaults.read_partitions cannot be negative"))
	}
	if s.Defaults.WriteConcurrency < 0 {
		errs = append(errs, errors.New("defaults.write_concurrency cannot be negative"))
	}

	return errors.Join(errs...)
}

func validateMode(m WriteMode) error {
	switch m {
	case "", ModeAppend, ModeTruncate, ModeCreateOnly, ModeUpsert, ModeReplaceAtomic:
		return nil
	default:
		return fmt.Errorf("unknown mode %q (want append, truncate, create_only, upsert or replace_atomic)", m)
	}
}

func (s *Spec) applyDefaults() {
	d := &s.Defaults
	if d.Mode == "" {
		d.Mode = ModeAppend
	}
	if d.BatchBytes == 0 {
		d.BatchBytes = 4 << 20
	}
	if d.WriteConcurrency == 0 {
		d.WriteConcurrency = min(16, max(4, runtime.NumCPU()))
	}
	if d.ReadPartitions == 0 {
		// Over-partition so that work stealing absorbs size skew between
		// partitions; see the pipeline's partition-affinity design.
		d.ReadPartitions = 6 * d.WriteConcurrency
	}
	if d.MemoryBudget == 0 {
		d.MemoryBudget = 1 << 30
	}
	if d.DeferIndexes == nil {
		t := true
		d.DeferIndexes = &t
	}

	m := &s.Mapping
	if m.Arrays == "" {
		m.Arrays = ArraysJSONB
	}
	if m.OnTypeConflict == "" {
		m.OnTypeConflict = ConflictJSON
	}
	if m.MinFieldFreq == 0 {
		m.MinFieldFreq = 0.02
	}
	if m.SpillColumn == nil {
		// Mandatory by default: an unseen field can then never make a document
		// un-migratable, which is the only honest answer to sampled inference.
		c := "_rest"
		m.SpillColumn = &c
	}
	if m.Flatten.MaxDepth == 0 {
		m.Flatten.MaxDepth = 3
	}
	if m.Flatten.Separator == "" {
		m.Flatten.Separator = "_"
	}
	if m.Flatten.Threshold == 0 {
		m.Flatten.Threshold = 0.9
	}
	if m.Flatten.Enabled == nil {
		t := true
		m.Flatten.Enabled = &t
	}

	for i := range s.Objects {
		if s.Objects[i].Mode == "" {
			s.Objects[i].Mode = d.Mode
		}
	}
}
