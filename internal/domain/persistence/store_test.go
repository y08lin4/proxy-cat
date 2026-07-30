package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	return NewStore(dir), dir
}

func TestSaveAndLoadProfile(t *testing.T) {
	store, _ := newTestStore(t)

	p := proxy.Profile{
		ID:   "p1",
		Name: "Test Profile",
		Proxies: []proxy.ProxyNode{
			{ID: "n1", Name: "HK 01", Type: "ss", Server: "hk.example.com", Port: 8388},
			{ID: "n2", Name: "US", Type: "vmess", Server: "us.example.com", Port: 443},
		},
		Settings: proxy.DefaultSettings(),
	}

	if err := store.SaveProfile(p); err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}

	loaded, err := store.LoadProfile("p1")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}

	if loaded.ID != p.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, p.ID)
	}
	if loaded.Name != p.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, p.Name)
	}
	if len(loaded.Proxies) != len(p.Proxies) {
		t.Fatalf("len(Proxies) = %d, want %d", len(loaded.Proxies), len(p.Proxies))
	}
	for i, node := range loaded.Proxies {
		if node.Name != p.Proxies[i].Name {
			t.Errorf("Proxies[%d].Name = %q, want %q", i, node.Name, p.Proxies[i].Name)
		}
	}
	if loaded.Settings.MixedPort != p.Settings.MixedPort {
		t.Errorf("Settings.MixedPort = %d, want %d", loaded.Settings.MixedPort, p.Settings.MixedPort)
	}
}

func TestSaveProfileRoundTripJSON(t *testing.T) {
	store, dir := newTestStore(t)

	p := proxy.Profile{
		ID:   "roundtrip",
		Name: "Round Trip",
		Proxies: []proxy.ProxyNode{
			{ID: "n1", Name: "HK", Type: "ss", Server: "hk.example.com", Port: 8388},
		},
		ProxyGroups: proxy.DefaultGroups([]string{"HK"}),
		Rules:       proxy.DefaultRules("PROXY"),
		Settings:    proxy.DefaultSettings(),
	}

	if err := store.SaveProfile(p); err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}

	// Verify the JSON file was written.
	path := filepath.Join(dir, "profiles", "roundtrip.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("JSON file is empty")
	}
	if !json.Valid(data) {
		t.Fatalf("JSON file is not valid JSON: %s", string(data))
	}

	loaded, err := store.LoadProfile("roundtrip")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}

	if loaded.ID != p.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, p.ID)
	}
	if loaded.Name != p.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, p.Name)
	}
	if len(loaded.Rules) != len(p.Rules) {
		t.Errorf("len(Rules) = %d, want %d", len(loaded.Rules), len(p.Rules))
	}
	if len(loaded.ProxyGroups) != len(p.ProxyGroups) {
		t.Errorf("len(ProxyGroups) = %d, want %d", len(loaded.ProxyGroups), len(p.ProxyGroups))
	}
}

func TestListProfiles(t *testing.T) {
	store, _ := newTestStore(t)

	profiles := []proxy.Profile{
		{ID: "a", Name: "Alpha", Proxies: []proxy.ProxyNode{{Name: "A1", Type: "ss", Server: "a.example.com", Port: 8388}}},
		{ID: "b", Name: "Beta", Proxies: []proxy.ProxyNode{{Name: "B1", Type: "vmess", Server: "b.example.com", Port: 443}}},
		{ID: "c", Name: "Gamma", Proxies: []proxy.ProxyNode{
			{Name: "C1", Type: "ss", Server: "c1.example.com", Port: 8388},
			{Name: "C2", Type: "trojan", Server: "c2.example.com", Port: 443},
		}},
	}

	for _, p := range profiles {
		if err := store.SaveProfile(p); err != nil {
			t.Fatalf("SaveProfile(%q) error = %v", p.ID, err)
		}
	}

	metas, err := store.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}

	if len(metas) != 3 {
		t.Fatalf("len(metas) = %d, want 3", len(metas))
	}

	metaByID := make(map[string]ProfileMeta, len(metas))
	for _, m := range metas {
		metaByID[m.ID] = m
	}

	for _, p := range profiles {
		m, ok := metaByID[p.ID]
		if !ok {
			t.Errorf("ListProfiles() missing profile %q", p.ID)
			continue
		}
		if m.Name != p.Name {
			t.Errorf("meta[%q].Name = %q, want %q", p.ID, m.Name, p.Name)
		}
		if m.ProxyCount != len(p.Proxies) {
			t.Errorf("meta[%q].ProxyCount = %d, want %d", p.ID, m.ProxyCount, len(p.Proxies))
		}
		if m.UpdatedAt.IsZero() {
			t.Errorf("meta[%q].UpdatedAt is zero", p.ID)
		}
	}
}

func TestDeleteProfile(t *testing.T) {
	store, dir := newTestStore(t)

	p := proxy.Profile{
		ID:   "del",
		Name: "Delete Me",
		Proxies: []proxy.ProxyNode{
			{Name: "HK", Type: "ss", Server: "hk.example.com", Port: 8388},
		},
	}
	if err := store.SaveProfile(p); err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}

	// Verify file exists.
	path := filepath.Join(dir, "profiles", "del.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat() before delete error = %v", err)
	}

	if err := store.DeleteProfile("del"); err != nil {
		t.Fatalf("DeleteProfile() error = %v", err)
	}

	// Verify file is gone.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, but Stat() returned: %v", err)
	}

	// Load should fail with ErrNotFound.
	if _, err := store.LoadProfile("del"); err != ErrNotFound {
		t.Errorf("LoadProfile() after delete: error = %v, want ErrNotFound", err)
	}

	// Deleting again should return ErrNotFound.
	if err := store.DeleteProfile("del"); err != ErrNotFound {
		t.Errorf("DeleteProfile() twice: error = %v, want ErrNotFound", err)
	}
}

func TestLoadProfileNotFound(t *testing.T) {
	store, _ := newTestStore(t)

	_, err := store.LoadProfile("nonexistent")
	if err != ErrNotFound {
		t.Errorf("LoadProfile() error = %v, want ErrNotFound", err)
	}
}

func TestListProfilesEmptyDir(t *testing.T) {
	store, dir := newTestStore(t)

	// Only the temp dir exists; no profiles/ subdirectory yet.
	metas, err := store.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}
	if len(metas) != 0 {
		t.Errorf("len(metas) = %d, want 0", len(metas))
	}
	_ = dir // ensure dir is referenced
}

func TestListProfilesIgnoresNonJSON(t *testing.T) {
	store, dir := newTestStore(t)

	// Save a valid profile first so the profiles/ directory exists.
	p := proxy.Profile{
		ID:   "ok",
		Name: "OK",
		Proxies: []proxy.ProxyNode{
			{Name: "HK", Type: "ss", Server: "hk.example.com", Port: 8388},
		},
	}
	if err := store.SaveProfile(p); err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}

	// Write a non-JSON file into the profiles directory.
	profilesDir := filepath.Join(dir, "profiles")
	if err := os.WriteFile(filepath.Join(profilesDir, "notes.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile(notes.txt) error = %v", err)
	}
	// Write a subdirectory.
	if err := os.MkdirAll(filepath.Join(profilesDir, "subdir"), 0o755); err != nil {
		t.Fatalf("MkdirAll(subdir) error = %v", err)
	}

	metas, err := store.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("len(metas) = %d, want 1", len(metas))
	}
	if metas[0].ID != "ok" {
		t.Errorf("metas[0].ID = %q, want %q", metas[0].ID, "ok")
	}
}
