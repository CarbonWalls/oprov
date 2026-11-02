package config

import "encoding/json"

// RootConfig is the typed view of the opencode config document. Only the
// fields we manage are modelled; everything else in the file is left
// untouched because edits are performed at the JSONC AST member level.
type RootConfig struct {
	Schema    string                     `json:"$schema,omitempty"`
	Providers map[string]*ProviderConfig `json:"providers,omitempty"`
	// Legacy single-provider map, preserved but not edited.
	Provider map[string]json.RawMessage `json:"provider,omitempty"`
}

// ProviderConfig mirrors opencode's `providers.<id>` shape for a custom
// provider. It is used for reading and to pre-fill forms; writes are done
// member-by-member on the AST so unrelated settings survive untouched.
type ProviderConfig struct {
	Name          string                  `json:"name,omitempty"`
	Env           []string                `json:"env,omitempty"`
	Package       string                  `json:"package,omitempty"`
	Canonical     string                  `json:"canonical,omitempty"`
	Compatibility json.RawMessage         `json:"compatibility,omitempty"`
	Settings      map[string]any          `json:"settings,omitempty"`
	Headers       map[string]string       `json:"headers,omitempty"`
	Body          map[string]any          `json:"body,omitempty"`
	Models        map[string]*ModelConfig `json:"models,omitempty"`
}

// ModelConfig mirrors opencode's `providers.<id>.models.<mid>` shape.
type ModelConfig struct {
	ModelID       string            `json:"modelID,omitempty"`
	Name          string            `json:"name,omitempty"`
	Family        string            `json:"family,omitempty"`
	Type          string            `json:"type,omitempty"`
	ReleaseDate   string            `json:"releaseDate,omitempty"`
	Package       string            `json:"package,omitempty"`
	Settings      map[string]any    `json:"settings,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          map[string]any    `json:"body,omitempty"`
	Capabilities  *Capabilities     `json:"capabilities,omitempty"`
	Compatibility json.RawMessage   `json:"compatibility,omitempty"`
	Variants      []json.RawMessage `json:"variants,omitempty"`
	Cost          *Cost             `json:"cost,omitempty"`
	Limit         *Limit            `json:"limit,omitempty"`
	Disabled      bool              `json:"disabled,omitempty"`
}

type Cost struct {
	Input  *float64           `json:"input,omitempty"`
	Output *float64           `json:"output,omitempty"`
	Cache  map[string]float64 `json:"cache,omitempty"`
}

type Limit struct {
	Context *int `json:"context,omitempty"`
	Input   *int `json:"input,omitempty"`
	Output  *int `json:"output,omitempty"`
}

type Capabilities struct {
	Input       []string   `json:"input,omitempty"`
	Output      []string   `json:"output,omitempty"`
	Tools       *bool      `json:"tools,omitempty"`
	Reasoning   *Reasoning `json:"reasoning,omitempty"`
	Temperature *bool      `json:"temperature,omitempty"`
}

type Reasoning struct {
	Supported *bool    `json:"supported,omitempty"`
	Effort    []string `json:"effort,omitempty"`
}

// KeyInfo describes the resolved state of a provider's API key.
type KeyInfo struct {
	Status  KeyStatus
	Source  string // human description, e.g. `{env:ACME_API_KEY}` or "keys/acme.key"
	VarName string // env var name when stored as {env:...}
	Path    string // file path when stored as {file:...}
	// Resolved is the actual credential value when available.
	Resolved string
	// Secret is true when Resolved holds a real secret (never print it).
	Secret bool
}

type KeyStatus int

const (
	KeyMissing KeyStatus = iota // no key configured at all
	KeyUnset                    // configured (env/file) but not resolvable right now
	KeySet                      // resolvable now
)
