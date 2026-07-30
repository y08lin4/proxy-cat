package persistence

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

var ErrNotFound = errors.New("profile not found")

type Store struct {
	dataDir string
}

func NewStore(dataDir string) *Store {
	return &Store{dataDir: dataDir}
}

type ProfileMeta struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	UpdatedAt  time.Time `json:"updatedAt"`
	ProxyCount int       `json:"proxyCount"`
}

func (s *Store) profilesDir() string {
	return filepath.Join(s.dataDir, "profiles")
}

func (s *Store) profilePath(id string) string {
	return filepath.Join(s.profilesDir(), id+".json")
}

func (s *Store) SaveProfile(p proxy.Profile) error {
	if err := os.MkdirAll(s.profilesDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.profilePath(p.ID), data, 0o600)
}

func (s *Store) LoadProfile(id string) (proxy.Profile, error) {
	path := s.profilePath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return proxy.Profile{}, ErrNotFound
		}
		return proxy.Profile{}, err
	}
	var p proxy.Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return proxy.Profile{}, err
	}
	return p, nil
}

func (s *Store) ListProfiles() ([]ProfileMeta, error) {
	dir := s.profilesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var metas []ProfileMeta
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var p proxy.Profile
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		metas = append(metas, ProfileMeta{
			ID:         p.ID,
			Name:       p.Name,
			UpdatedAt:  info.ModTime(),
			ProxyCount: len(p.Proxies),
		})
	}
	return metas, nil
}

func (s *Store) DeleteProfile(id string) error {
	path := s.profilePath(id)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}
		return err
	}
	return os.Remove(path)
}
