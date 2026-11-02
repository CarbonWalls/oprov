package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Key resolution + storage helpers. opencode resolves credentials from a
// provider's settings.apiKey which may be a literal, a {env:VAR} reference or
// a {file:path} reference; failing that it uses the provider's `env` list.

// KeySource describes how a key is stored.
type KeySource string

const (
	SourceNone    KeySource = "none"   // no key configured
	SourceLiteral KeySource = "inline" // key stored inline in the config
	SourceEnv     KeySource = "env"    // {env:VAR} reference
	SourceFile    KeySource = "file"   // {file:path} reference
)

var (
	envRefRe  = regexp.MustCompile(`^\{env:([A-Za-z_][A-Za-z0-9_]*)\}$`)
	fileRefRe = regexp.MustCompile(`^\{file:(.*)\}$`)
)

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		if p == "~" {
			return home
		}
		return filepath.Join(home, p[2:])
	}
	return p
}

// KeyInfo resolves the current key state of a provider config.
func KeyInfoOf(p *ProviderConfig) KeyInfo {
	info := KeyInfo{Status: KeyMissing}
	if p == nil {
		return info
	}
	apiKey, _ := p.Settings["apiKey"].(string)
	apiKey = strings.TrimSpace(apiKey)
	if apiKey != "" {
		switch {
		case envRefRe.MatchString(apiKey):
			m := envRefRe.FindStringSubmatch(apiKey)
			info.VarName = m[1]
			info.Source = fmt.Sprintf("{env:%s}", m[1])
			if v, ok := os.LookupEnv(m[1]); ok && v != "" {
				info.Status = KeySet
				info.Resolved = v
				info.Secret = true
			} else {
				info.Status = KeyUnset
			}
			return info
		case fileRefRe.MatchString(apiKey):
			m := fileRefRe.FindStringSubmatch(apiKey)
			info.Path = m[1]
			info.Source = fmt.Sprintf("{file:%s}", m[1])
			if v, err := os.ReadFile(expandHome(m[1])); err == nil && len(strings.TrimSpace(string(v))) > 0 {
				info.Status = KeySet
				info.Resolved = strings.TrimSpace(string(v))
				info.Secret = true
			} else {
				info.Status = KeyUnset
			}
			return info
		default:
			info.Source = "inline"
			info.Status = KeySet
			info.Resolved = apiKey
			info.Secret = true
			return info
		}
	}
	// Fall back to the provider env list.
	for _, v := range p.Env {
		if val, ok := os.LookupEnv(v); ok && val != "" {
			info.VarName = v
			info.Source = fmt.Sprintf("{env:%s}", v)
			info.Status = KeySet
			info.Resolved = val
			info.Secret = true
			return info
		}
	}
	if len(p.Env) > 0 {
		info.Source = fmt.Sprintf("env:%s", strings.Join(p.Env, ","))
		info.Status = KeyUnset
	}
	return info
}

// KeyStorage describes where a newly provided key should be written.
type KeyStorage struct {
	Source KeySource
	// EnvVar is the variable name for SourceEnv.
	EnvVar string
	// FilePath is the file path for SourceFile.
	FilePath string
}

// DefaultKeyFile returns the on-disk location used for {file:...} storage,
// adjacent to the config document in a `keys/` directory.
func DefaultKeyFile(configPath, providerID string) string {
	return filepath.Join(filepath.Dir(configPath), "keys", providerID+".key")
}

// SuggestEnvVar derives a conventional variable name from a provider ID.
func SuggestEnvVar(providerID string) string {
	re := regexp.MustCompile(`[^A-Za-z0-9_]+`)
	up := strings.ToUpper(re.ReplaceAllString(providerID, "_"))
	up = strings.Trim(up, "_")
	if up == "" {
		up = "API_KEY"
	}
	return up + "_API_KEY"
}

// WriteFileKey stores the raw secret in path with 0600 permissions.
func WriteFileKey(path, secret string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(secret+"\n"), 0o600); err != nil {
		return err
	}
	return nil
}

// RemoveKeyFile deletes a stored key file (ignoring errors if absent).
func RemoveKeyFile(path string) bool {
	if path == "" {
		return false
	}
	if err := os.Remove(expandHome(path)); err == nil {
		return true
	}
	return false
}

// ShellConfig describes an export line to append to a shell rc file.
type ShellConfig struct {
	File string
	Line string
}

// rcFileFor detects the user's shell rc file from $SHELL.
func rcFileFor() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	shell := strings.TrimSpace(os.Getenv("SHELL"))
	switch filepath.Base(shell) {
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "bash":
		return filepath.Join(home, ".bashrc")
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	}
	return filepath.Join(home, ".bashrc")
}

// ShellExportsFor builds the rc edit needed to make `var` available, or nil
// when it is already exported.
func ShellExportsFor(providerID, varName, secret string) *ShellConfig {
	file := rcFileFor()
	line := fmt.Sprintf("export %s='%s'", varName, strings.ReplaceAll(secret, "'", `'\''`))
	if existing, err := os.ReadFile(file); err == nil {
		for _, l := range strings.Split(string(existing), "\n") {
			trimmed := strings.TrimSpace(l)
			if strings.HasPrefix(trimmed, "export "+varName+"=") {
				return nil // already exported
			}
		}
	}
	return &ShellConfig{
		File: file,
		Line: fmt.Sprintf("# opencode provider key for %s (managed by oprov)\n%s", providerID, line),
	}
}

// ApplyShellExport appends the export to the rc file.
func ApplyShellExport(sc *ShellConfig) error {
	if sc == nil {
		return nil
	}
	f, err := os.OpenFile(sc.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString("\n" + sc.Line + "\n"); err != nil {
		return err
	}
	return nil
}
