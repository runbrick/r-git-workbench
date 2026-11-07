package settings

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type App struct {
	Language   string            `json:"language,omitempty"`
	PullPolicy string            `json:"pull_policy,omitempty"`
	Keymap     map[string]string `json:"keymap,omitempty"`
	Recent     []string          `json:"recent,omitempty"`
}

type Repository struct {
	ChangesSplit     float32           `json:"changes_split,omitempty"`
	LogBranchesSplit float32           `json:"log_branches_split,omitempty"`
	LogDetailsSplit  float32           `json:"log_details_split,omitempty"`
	TreeMode         bool              `json:"tree_mode,omitempty"`
	LogFilter        map[string]string `json:"log_filter,omitempty"`
	PanelVisible     map[string]bool   `json:"panel_visible,omitempty"`
}

type Store struct {
	Root string
}

func User() (*Store, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return &Store{Root: filepath.Join(root, "r-git-workbench")}, nil
}

func (store *Store) LoadApp() (App, error) {
	var value App
	err := readJSON(filepath.Join(store.Root, "settings.json"), &value)
	return value, err
}

func (store *Store) SaveApp(value App) error {
	return writeJSON(filepath.Join(store.Root, "settings.json"), value)
}

func (store *Store) LoadRepository(root string) (Repository, error) {
	var value Repository
	err := readJSON(store.repositoryPath(root), &value)
	return value, err
}

func (store *Store) SaveRepository(root string, value Repository) error {
	if strings.TrimSpace(root) == "" {
		return errors.New("repository path is required")
	}
	return writeJSON(store.repositoryPath(root), value)
}

func (store *Store) repositoryPath(root string) string {
	path, _ := filepath.Abs(filepath.Clean(root))
	sum := sha256.Sum256([]byte(strings.ToLower(path)))
	return filepath.Join(store.Root, "repos", hex.EncodeToString(sum[:])+".json")
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".settings-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
