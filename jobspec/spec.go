package jobspec

import "time"

// Spec is a migration job as written on disk. It is meant to be committed to a
// repository, so it holds references to credentials, never credentials.
//
// Fields decoded from YAML keep their raw text; resolved values live on the
// Resolved struct that Load returns alongside it.
type Spec struct {
	Version int    `yaml:"version"`
	Name    string `yaml:"name"`

	Source Endpoint `yaml:"source"`
	Sink   Endpoint `yaml:"sink"`

	Defaults      Defaults      `yaml:"defaults"`
	Mapping       Mapping       `yaml:"mapping"`
	Objects       []Object      `yaml:"objects"`
	Capabilities  Capabilities  `yaml:"capabilities"`
	Observability Observability `yaml:"observability"`
}

// Endpoint is one side of a migration.
type Endpoint struct {
	// DSN is the raw, uninterpolated connection string, normally a reference
	// such as ${env:SRC_DSN}. Load rejects a literal credential here.
	DSN string `yaml:"dsn"`

	// Options are driver-specific and validated by the driver, not here.
	Options map[string]any `yaml:"options"`
}

// Defaults apply to every object unless the object overrides them.
type Defaults struct {
	Mode             WriteMode `yaml:"mode"`
	BatchBytes       ByteSize  `yaml:"batch_bytes"`
	ReadPartitions   int       `yaml:"read_partitions"`
	WriteConcurrency int       `yaml:"write_concurrency"`
	DeferIndexes     *bool     `yaml:"defer_indexes"`
	MemoryBudget     ByteSize  `yaml:"memory_budget"`
}

// WriteMode is how a target object is populated.
type WriteMode string

const (
	// ModeAppend writes into whatever is already there.
	ModeAppend WriteMode = "append"
	// ModeTruncate empties the target first.
	ModeTruncate WriteMode = "truncate"
	// ModeCreateOnly fails if the target already exists.
	ModeCreateOnly WriteMode = "create_only"
	// ModeUpsert merges on primary key.
	ModeUpsert WriteMode = "upsert"
	// ModeReplaceAtomic loads into a temporary object and swaps it in with a
	// single rename, so a re-run never leaves the target half-populated.
	ModeReplaceAtomic WriteMode = "replace_atomic"
)

// Mapping holds the schema-translation policy. Phase 4 consumes this; it is
// declared here so specs written today remain valid.
type Mapping struct {
	Flatten        Flatten           `yaml:"flatten"`
	Arrays         ArrayPolicy       `yaml:"arrays"`
	OnTypeConflict ConflictPolicy    `yaml:"on_type_conflict"`
	MinFieldFreq   float64           `yaml:"min_field_frequency"`
	SpillColumn    *string           `yaml:"spill_column"`
	Naming         string            `yaml:"naming"`
	AssumeTimezone string            `yaml:"assume_timezone"`
	OnLoss         map[string]string `yaml:"on_loss"`
	MySQL          map[string]any    `yaml:"mysql"`
	Mongo          map[string]any    `yaml:"mongo"`
}

type Flatten struct {
	Enabled   *bool   `yaml:"enabled"`
	MaxDepth  int     `yaml:"max_depth"`
	Separator string  `yaml:"separator"`
	Threshold float64 `yaml:"threshold"`
}

type ArrayPolicy string

const (
	ArraysJSONB     ArrayPolicy = "jsonb"
	ArraysNative    ArrayPolicy = "native"
	ArraysNormalize ArrayPolicy = "normalize"
)

type ConflictPolicy string

const (
	ConflictJSON   ConflictPolicy = "json"
	ConflictText   ConflictPolicy = "text"
	ConflictWidest ConflictPolicy = "widest"
	ConflictFail   ConflictPolicy = "fail"
)

// Object is one source table or collection and where it lands.
type Object struct {
	Source         string `yaml:"source"`
	SourcePattern  string `yaml:"source_pattern"`
	Target         string `yaml:"target"`
	TargetTemplate string `yaml:"target_template"`

	Mode         WriteMode                `yaml:"mode"`
	PartitionKey string                   `yaml:"partition_key"`
	Filter       string                   `yaml:"filter"`
	Fields       map[string]FieldOverride `yaml:"fields"`
}

// FieldOverride escapes bad schema inference without touching Go.
type FieldOverride struct {
	Type     string `yaml:"type"`
	Nullable *bool  `yaml:"nullable"`
	Rename   string `yaml:"rename"`
	Skip     bool   `yaml:"skip"`
}

// Capabilities lets an operator override provider detection when it gets a
// cluster wrong, without waiting for a new binary.
type Capabilities struct {
	Force map[string]bool `yaml:"force"`
}

type Observability struct {
	Progress    string        `yaml:"progress"`
	DeadLetter  string        `yaml:"dead_letter"`
	MaxErrors   int           `yaml:"max_errors"`
	Checkpoint  string        `yaml:"checkpoint"`
	LogInterval time.Duration `yaml:"log_interval"`
}

// Resolved carries the values that only exist after interpolation. It is kept
// separate from Spec so that Spec stays safe to marshal back out.
type Resolved struct {
	Source DSN
	Sink   DSN
}
