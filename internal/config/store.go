package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"
)

// Store keeps an opencode config document open for editing. The file is kept
// as a HuJSON (JSONC) syntax tree so comments, trailing commas and unrelated
// settings survive every edit. Writes are performed member-by-member on the
// tree and persisted atomically.
type Store struct {
	Path string
	tree hujson.Value
}

const schemaURL = "https://opencode.ai/config.json"

// GlobalCandidates returns the possible global config locations in the order
// we prefer them, honouring XDG_CONFIG_HOME.
func GlobalCandidates() []string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "~"
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "opencode")
	return []string{
		filepath.Join(dir, "opencode.jsonc"),
		filepath.Join(dir, "opencode.json"),
	}
}

// ResolveConfigPath picks the global config file to manage; the second
// return value reports whether the file already exists.
func ResolveConfigPath() (string, bool) {
	for _, p := range GlobalCandidates() {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, true
		}
	}
	return GlobalCandidates()[0], false
}

// permitJSONC reports whether the target file should be written as JSONC.
func useJSONC(path string) bool {
	return strings.HasSuffix(path, ".jsonc")
}

// Load reads and parses the config at path (or creates an in-memory empty
// document when the file does not exist yet).
func Load(path string) (*Store, error) {
	s := &Store{Path: path}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Fresh document with the common scaffolding.
			src := fmt.Sprintf("{\n  \"$schema\": %q,\n  \"providers\": {}\n}\n", schemaURL)
			tree, perr := hujson.Parse([]byte(src))
			if perr != nil {
				return nil, perr
			}
			s.tree = tree
			return s, nil
		}
		return nil, err
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "{}" {
		raw = []byte(fmt.Sprintf("{\n  \"$schema\": %q,\n  \"providers\": {}\n}\n", schemaURL))
	}
	tree, err := hujson.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	s.tree = tree
	// Make sure a top-level object exists.
	if tree.Value.(hujson.ValueTrimmed).Kind() != '{' {
		return nil, fmt.Errorf("parse %s: top-level value is not an object", path)
	}
	return s, nil
}

// --- AST helpers -----------------------------------------------------------

func memberName(m *hujson.ObjectMember) string {
	lit, ok := m.Name.Value.(hujson.Literal)
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(lit, &s) != nil {
		return ""
	}
	return s
}

// objPtr extracts the *Object of a value that is a JSON object, or nil.
func objPtr(v *hujson.Value) *hujson.Object {
	if v == nil {
		return nil
	}
	o, ok := v.Value.(*hujson.Object)
	if !ok {
		return nil
	}
	return o
}

// getChild returns the value of object member `name` (pointer to the member's
// Value, so in-place edits work), or nil when absent or when owner is not an
// object.
func getChild(owner *hujson.Value, name string) *hujson.Value {
	o := objPtr(owner)
	if o == nil {
		return nil
	}
	for i := range o.Members {
		if memberName(&o.Members[i]) == name {
			return &o.Members[i].Value
		}
	}
	return nil
}

// setChild replaces `name` in owner with the given standard-JSON bytes, or
// appends it when missing. New members get a leading newline so hujson's
// formatter expands them on their own line. It returns an error when owner
// is not an object.
func setChild(owner *hujson.Value, name string, raw []byte) error {
	o := objPtr(owner)
	if o == nil {
		return fmt.Errorf("%q is not an object", name)
	}
	nv, err := hujson.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid value for %q: %w", name, err)
	}
	for i := range o.Members {
		if memberName(&o.Members[i]) == name {
			o.Members[i].Value = nv
			return nil
		}
	}
	o.Members = append(o.Members, hujson.ObjectMember{
		Name:  hujson.Value{BeforeExtra: hujson.Extra("\n"), Value: hujson.String(name)},
		Value: nv,
	})
	return nil
}

func removeChild(owner *hujson.Value, name string) bool {
	o := objPtr(owner)
	if o == nil {
		return false
	}
	for i := range o.Members {
		if memberName(&o.Members[i]) == name {
			o.Members = append(o.Members[:i], o.Members[i+1:]...)
			return true
		}
	}
	return false
}

func stdJSON(v *hujson.Value) []byte {
	if v == nil {
		return nil
	}
	b, err := hujson.Standardize(v.Pack())
	if err != nil {
		return b
	}
	return b
}

// --- Public API ------------------------------------------------------------

// EnsureProviders returns the `providers` object value, creating it when
// missing (e.g. legacy "provider" config that had none).
func (s *Store) EnsureProviders() *hujson.Value {
	root := &s.tree
	providers := getChild(root, "providers")
	if providers == nil {
		_ = setChild(root, "providers", []byte("{}"))
		providers = getChild(root, "providers")
	}
	return providers
}

// ProviderIDs returns provider IDs in document order.
func (s *Store) ProviderIDs() []string {
	var ids []string
	o := objPtr(s.EnsureProviders())
	if o == nil {
		return ids
	}
	for i := range o.Members {
		id := memberName(&o.Members[i])
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// Provider returns the provider value for id, or nil.
func (s *Store) Provider(id string) *hujson.Value {
	return getChild(s.EnsureProviders(), id)
}

// ReadProvider decodes a provider into its typed form. Missing provider
// yields a zero ProviderConfig.
func (s *Store) ReadProvider(id string) ProviderConfig {
	var p ProviderConfig
	v := s.Provider(id)
	if v == nil {
		return p
	}
	_ = json.Unmarshal(stdJSON(v), &p)
	return p
}

// Model returns the model value for provider/model, or nil.
func (s *Store) Model(providerID, modelID string) *hujson.Value {
	p := s.Provider(providerID)
	if p == nil {
		return nil
	}
	return getChild(p, "models")
}

// ModelIDs returns model IDs in document order.
func (s *Store) ModelIDs(providerID string) []string {
	var ids []string
	models := getChild(s.Provider(providerID), "models")
	o := objPtr(models)
	if o == nil {
		return ids
	}
	for i := range o.Members {
		if id := memberName(&o.Members[i]); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// ReadModel decodes a single model.
func (s *Store) ReadModel(providerID, modelID string) ModelConfig {
	var m ModelConfig
	v := s.Model(providerID, modelID)
	if v == nil {
		return m
	}
	_ = json.Unmarshal(stdJSON(v), &m)
	return m
}

// --- Mutations -------------------------------------------------------------

func (s *Store) setProviderField(id, field string, value any) error {
	p := s.Provider(id)
	if p == nil {
		return fmt.Errorf("provider %q does not exist", id)
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return setChild(p, field, b)
}

// AddProvider inserts a new provider (with its typed map of models), or
// replaces it when it already exists.
func (s *Store) AddProvider(id string, p ProviderConfig) error {
	providers := s.EnsureProviders()
	o := objPtr(providers)
	if o == nil {
		return fmt.Errorf("providers is not an object")
	}
	body, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	nv, err := hujson.Parse(body)
	if err != nil {
		return err
	}
	for i := range o.Members {
		if memberName(&o.Members[i]) == id {
			o.Members[i].Value = nv
			return nil
		}
	}
	o.Members = append(o.Members, hujson.ObjectMember{
		Name:  hujson.Value{BeforeExtra: hujson.Extra("\n"), Value: hujson.String(id)},
		Value: nv,
	})
	return nil
}

// DeleteProvider removes a provider entirely.
func (s *Store) DeleteProvider(id string) bool {
	return removeChild(s.EnsureProviders(), id)
}

func (s *Store) SetProviderName(id, name string) error {
	if name == "" {
		removeChild(s.Provider(id), "name")
		return nil
	}
	return s.setProviderField(id, "name", name)
}

func (s *Store) SetProviderPackage(id, pkg string) error {
	if pkg == "" {
		removeChild(s.Provider(id), "package")
		return nil
	}
	return s.setProviderField(id, "package", pkg)
}

func (s *Store) SetProviderBaseURL(id, url string) error {
	p := s.Provider(id)
	if p == nil {
		return fmt.Errorf("provider %q does not exist", id)
	}
	settings := getChild(p, "settings")
	if settings == nil {
		if err := setChild(p, "settings", []byte("{}")); err != nil {
			return err
		}
		settings = getChild(p, "settings")
	}
	if url == "" {
		removeChild(settings, "baseURL")
		return nil
	}
	return setChild(settings, "baseURL", mustMarshal(url))
}

func (s *Store) SetProviderEnv(id string, vars []string) error {
	p := s.Provider(id)
	if p == nil {
		return fmt.Errorf("provider %q does not exist", id)
	}
	if len(vars) == 0 {
		removeChild(p, "env")
		return nil
	}
	return setChild(p, "env", mustMarshal(vars))
}

// SetProviderAPIKey sets settings.apiKey (deleting it when key is "").
func (s *Store) SetProviderAPIKey(id, key string) error {
	p := s.Provider(id)
	if p == nil {
		return fmt.Errorf("provider %q does not exist", id)
	}
	settings := getChild(p, "settings")
	if settings == nil {
		if err := setChild(p, "settings", []byte("{}")); err != nil {
			return err
		}
		settings = getChild(p, "settings")
	}
	if key == "" {
		removeChild(settings, "apiKey")
		return nil
	}
	return setChild(settings, "apiKey", mustMarshal(key))
}

// AddModel inserts or replaces a model on a provider.
func (s *Store) AddModel(providerID string, modelID string, m ModelConfig) error {
	p := s.Provider(providerID)
	if p == nil {
		return fmt.Errorf("provider %q does not exist", providerID)
	}
	models := getChild(p, "models")
	if models == nil {
		if err := setChild(p, "models", []byte("{}")); err != nil {
			return err
		}
		models = getChild(p, "models")
	}
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	nv, err := hujson.Parse(body)
	if err != nil {
		return err
	}
	o := objPtr(models)
	for i := range o.Members {
		if memberName(&o.Members[i]) == modelID {
			o.Members[i].Value = nv
			return nil
		}
	}
	o.Members = append(o.Members, hujson.ObjectMember{
		Name:  hujson.Value{BeforeExtra: hujson.Extra("\n"), Value: hujson.String(modelID)},
		Value: nv,
	})
	return nil
}

func (s *Store) DeleteModel(providerID, modelID string) bool {
	p := s.Provider(providerID)
	if p == nil {
		return false
	}
	models := getChild(p, "models")
	return removeChild(models, modelID)
}

// Save writes the document atomically, preserving JSONC when the path ends
// with .jsonc (otherwise formatted as standard JSON).
func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(s.Path); err == nil {
		mode = fi.Mode().Perm()
	}
	s.tree.Format()
	out := s.tree.Pack()
	if !useJSONC(s.Path) {
		// Strip comments / trailing commas so the result is strict JSON.
		std, err := hujson.Standardize(out)
		if err != nil {
			return err
		}
		std, err = hujson.Format(std)
		if err != nil {
			return err
		}
		out = std
	}
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, out, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
