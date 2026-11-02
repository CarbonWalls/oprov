package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `{
  "$schema": "https://opencode.ai/config.json",
  // keep this comment
  "model": "mercury/mercury-3",
  "providers": {
    "mercury": {
      "name": "retard", // trailing comment
      "package": "@opencode/ai/providers/openai-compatible",
      "env": ["MY_CUSTOM_API_KEY"],
      "settings": {
        "baseURL": "https://api.inceptionlabs.ai/v1",
        "apiKey": "{env:MY_CUSTOM_API_KEY}"
      },
      "models": {
        "mercury-3": {
          "name": "mercury",
          "limit": { "context": 1000000, "output": 131072 },
          "cost": { "input": 40, "output": 50 }
        }
      }
    }
  },
}
`

func writeSample(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadAndRoundTrip(t *testing.T) {
	p := writeSample(t, "opencode.jsonc")
	s, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	ids := s.ProviderIDs()
	if len(ids) != 1 || ids[0] != "mercury" {
		t.Fatalf("provider ids = %v", ids)
	}
	prov := s.ReadProvider("mercury")
	if prov.Name != "retard" || prov.Package == "" {
		t.Fatalf("bad provider: %+v", prov)
	}
	if prov.Settings["baseURL"] != "https://api.inceptionlabs.ai/v1" {
		t.Fatalf("baseURL = %v", prov.Settings["baseURL"])
	}
	if !strings.Contains(prov.Settings["apiKey"].(string), "env:MY_CUSTOM_API_KEY") {
		t.Fatalf("apiKey = %v", prov.Settings["apiKey"])
	}
	if got := s.ModelIDs("mercury"); len(got) != 1 || got[0] != "mercury-3" {
		t.Fatalf("model ids = %v", got)
	}
	// round trip: comments must survive
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if !strings.Contains(string(raw), "keep this comment") {
		t.Fatal("comment lost on round trip")
	}
	if !strings.Contains(string(raw), "trailing comment") {
		t.Fatal("trailing comment lost on round trip")
	}
}

func TestAddEditDeleteProvider(t *testing.T) {
	p := writeSample(t, "opencode.jsonc")
	s, _ := Load(p)

	// add
	pc := ProviderConfig{
		Name:    "Acme",
		Package: "@opencode/ai/providers/openai-compatible",
		Env:     []string{"ACME_API_KEY"},
		Settings: map[string]any{
			"baseURL": "https://llm.acme.example/v1",
		},
	}
	if err := s.AddProvider("acme", pc); err != nil {
		t.Fatal(err)
	}
	if got := s.ProviderIDs(); len(got) != 2 || got[1] != "acme" {
		t.Fatalf("after add: %v", got)
	}

	// edit
	if err := s.SetProviderBaseURL("acme", "https://new.example/v1"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetProviderEnv("acme", []string{"ACME_API_KEY", "ACME_ALT"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetProviderName("acme", "Acme Inc"); err != nil {
		t.Fatal(err)
	}

	// models
	m := ModelConfig{
		Name:    "Qwen 3 Coder",
		Limit:   &Limit{Context: intPtr(200000)},
		Cost:    &Cost{Input: floatPtr(0.5), Output: floatPtr(1.5)},
		ModelID: "qwen3-coder",
	}
	if err := s.AddModel("acme", "qwen3-coder", m); err != nil {
		t.Fatal(err)
	}
	if err := s.AddModel("acme", "agent/model-x", m); err != nil { // id with slash
		t.Fatal(err)
	}
	if got := s.ModelIDs("acme"); len(got) != 2 {
		t.Fatalf("models = %v", got)
	}

	// key via file ref
	if err := s.SetProviderAPIKey("acme", "{file:/tmp/key.acme}"); err != nil {
		t.Fatal(err)
	}
	// models must be inside acme, not inside mercury
	if got := s.ModelIDs("mercury"); len(got) != 1 {
		t.Fatalf("mercury models mutated: %v", got)
	}

	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if !strings.Contains(string(raw), "acme") || !strings.Contains(string(raw), "agent/model-x") {
		t.Fatalf("saved content missing additions:\n%s", raw)
	}

	// delete model, provider
	if !s.DeleteModel("acme", "agent/model-x") {
		t.Fatal("delete model failed")
	}
	if !s.DeleteProvider("acme") {
		t.Fatal("delete provider failed")
	}
	if len(s.ProviderIDs()) != 1 {
		t.Fatalf("providers after delete: %v", s.ProviderIDs())
	}
	_ = s.Save()
}

func TestAddProviderToFreshFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "opencode.json")
	s, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddProvider("x", ProviderConfig{Name: "X", Package: "pkg"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	// newly added providers must be expanded on their own line
	if !strings.Contains(string(raw), "\n\t\"providers\": {\n\t\t\"x\": {") &&
		!strings.Contains(string(raw), "\n  \"providers\": {\n    \"x\": {") {
		t.Fatalf("new provider not nicely formatted:\n%s", raw)
	}
	// strict JSON output — no comment markers (a lone `//` inside the
	// $schema URL is fine; comments look like `// ` or `/*`)
	if strings.Contains(string(raw), "// ") || strings.Contains(string(raw), "/*") {
		t.Fatal("strict JSON contains comment marker")
	}
}

func TestEnvKeyResolution(t *testing.T) {
	p := writeSample(t, "opencode.jsonc")
	s, _ := Load(p)
	prov := s.ReadProvider("mercury")
	info := KeyInfoOf(&prov)
	if info.Status != KeyMissing && info.Status != KeyUnset {
		t.Fatalf("status = %v", info.Status)
	}
	if info.VarName != "MY_CUSTOM_API_KEY" && info.Source != "{env:MY_CUSTOM_API_KEY}" {
		t.Fatalf("source = %v", info.Source)
	}

	t.Setenv("MY_CUSTOM_API_KEY", "sk-test")
	prov2 := s.ReadProvider("mercury")
	info2 := KeyInfoOf(&prov2)
	if info2.Status != KeySet || info2.Resolved != "sk-test" {
		t.Fatalf("resolved = %+v", info2)
	}
}

func TestFileKeyResolution(t *testing.T) {
	dir := t.TempDir()
	kf := filepath.Join(dir, "key.file")
	os.WriteFile(kf, []byte("secret-value\n"), 0o600)
	s := &Store{}
	p := ProviderConfig{Settings: map[string]any{"apiKey": "{file:" + kf + "}"}}
	info := KeyInfoOf(&p)
	if info.Status != KeySet || info.Resolved != "secret-value" {
		t.Fatalf("file key: %+v", info)
	}
	_ = s
}

func TestKeyFileHelpers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys", "acme.key")
	if err := WriteFileKey(path, "sekret"); err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(path)
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %o", fi.Mode().Perm())
	}
	if !RemoveKeyFile(path) {
		t.Fatal("RemoveKeyFile returned false")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file still exists")
	}
}

func intPtr(n int) *int           { return &n }
func floatPtr(f float64) *float64 { return &f }
