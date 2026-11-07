package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorePersistsAppAndRepositoryStateSeparately(t *testing.T) {
	store := &Store{Root: t.TempDir()}
	wantApp := App{Language: "zh", PullPolicy: "rebase", Keymap: map[string]string{"view.log": "Ctrl+9"}}
	if err := store.SaveApp(wantApp); err != nil {
		t.Fatal(err)
	}
	gotApp, err := store.LoadApp()
	if err != nil || gotApp.Language != wantApp.Language || gotApp.PullPolicy != wantApp.PullPolicy || gotApp.Keymap["view.log"] != "Ctrl+9" {
		t.Fatalf("app settings: %+v, %v", gotApp, err)
	}
	rootOne := filepath.Join(t.TempDir(), "repo-one")
	rootTwo := filepath.Join(t.TempDir(), "repo-two")
	wantRepo := Repository{ChangesSplit: 320, TreeMode: true, PanelVisible: map[string]bool{"log-details": false}}
	if err := store.SaveRepository(rootOne, wantRepo); err != nil {
		t.Fatal(err)
	}
	gotRepo, err := store.LoadRepository(rootOne)
	if err != nil || gotRepo.ChangesSplit != 320 || !gotRepo.TreeMode || gotRepo.PanelVisible["log-details"] {
		t.Fatalf("repository settings: %+v, %v", gotRepo, err)
	}
	otherRepo, err := store.LoadRepository(rootTwo)
	if err != nil || otherRepo.ChangesSplit != 0 {
		t.Fatalf("repository state leaked: %+v, %v", otherRepo, err)
	}
	entries, err := os.ReadDir(filepath.Join(store.Root, "repos"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("repository settings files: %v, %v", entries, err)
	}
	if info, err := os.Stat(filepath.Join(store.Root, "settings.json")); err != nil || info.Size() == 0 {
		t.Fatalf("settings file: %v, %v", info, err)
	}
}
