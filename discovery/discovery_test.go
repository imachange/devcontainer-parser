package discovery

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func touch(t *testing.T, root string, rel ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{root}, rel...)...)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFindPriority(t *testing.T) {
	root := t.TempDir()
	p3 := touch(t, root, ".devcontainer.json")
	p2b := touch(t, root, ".devcontainer", "b", "devcontainer.json")
	p2a := touch(t, root, ".devcontainer", "a", "devcontainer.json")
	p1 := touch(t, root, ".devcontainer", "devcontainer.json")

	got, err := Find(root)
	if err != nil || got != p1 {
		t.Fatalf("1 が最優先: got %q, err=%v, want %q", got, err, p1)
	}

	all, err := Candidates(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{p1, p2a, p2b, p3}; !reflect.DeepEqual(all, want) {
		t.Errorf("Candidates = %v, want %v", all, want)
	}

	if err := os.Remove(p1); err != nil {
		t.Fatal(err)
	}
	if got, err := Find(root); err != nil || got != p2a {
		t.Errorf("2 (辞書順の先頭): got %q, err=%v, want %q", got, err, p2a)
	}

	if err := os.Remove(p2a); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p2b); err != nil {
		t.Fatal(err)
	}
	if got, err := Find(root); err != nil || got != p3 {
		t.Errorf("3: got %q, err=%v, want %q", got, err, p3)
	}
}

func TestFindOnlyOneLevelDeep(t *testing.T) {
	root := t.TempDir()
	touch(t, root, ".devcontainer", "a", "b", "devcontainer.json")
	if got, err := Find(root); !errors.Is(err, ErrNotFound) {
		t.Errorf("2 階層下は、対象外: got %q, err=%v", got, err)
	}
}

func TestFindIgnoresOtherFiles(t *testing.T) {
	root := t.TempDir()
	touch(t, root, ".devcontainer", "Dockerfile")
	touch(t, root, ".devcontainer", "script", "bootstrap.sh")
	touch(t, root, "devcontainer.json") // ルート直下の devcontainer.json (先頭にドットが無い) は、対象外
	if got, err := Find(root); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %q, err=%v", got, err)
	}
}

func TestFindDirectoryNamedDevcontainerJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".devcontainer", "devcontainer.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".devcontainer.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := Find(root); !errors.Is(err, ErrNotFound) {
		t.Errorf("ディレクトリは、ファイルとして扱わない: got %q, err=%v", got, err)
	}
}

func TestFindNotFound(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
	}{
		{"空のディレクトリ", func(t *testing.T, root string) {}},
		{".devcontainer が空", func(t *testing.T, root string) {
			t.Helper()
			if err := os.MkdirAll(filepath.Join(root, ".devcontainer"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{".devcontainer がファイル", func(t *testing.T, root string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(root, ".devcontainer"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)
			got, err := Find(root)
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("got %q, err=%v", got, err)
			}
			all, err := Candidates(root)
			if err != nil || len(all) != 0 {
				t.Errorf("Candidates = %v, err=%v", all, err)
			}
		})
	}
}

func TestFindNonexistentRoot(t *testing.T) {
	if got, err := Find(filepath.Join(t.TempDir(), "nope")); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %q, err=%v", got, err)
	}
}

func TestPatternsMatchTheDocumentedOrder(t *testing.T) {
	want := []string{
		".devcontainer/devcontainer.json",
		".devcontainer/<DIR>/devcontainer.json",
		".devcontainer.json",
	}
	if !reflect.DeepEqual(Patterns, want) {
		t.Errorf("Patterns = %v", Patterns)
	}
}
