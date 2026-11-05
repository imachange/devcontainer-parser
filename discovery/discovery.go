// Package discovery は、ディレクトリから devcontainer.json を、規定の優先順位で探索する。
package discovery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotFound は、devcontainer.json が見つからなかったことを表す。
var ErrNotFound = errors.New("devcontainer.json not found")

// Patterns は、探す場所を、優先順位の順に並べたもの (エラーメッセージ用)。
var Patterns = []string{
	".devcontainer/devcontainer.json",
	".devcontainer/<DIR>/devcontainer.json",
	".devcontainer.json",
}

// Candidates は、root の下に存在する devcontainer.json を、優先順位の順にすべて返す。
//
//  1. .devcontainer/devcontainer.json
//  2. .devcontainer/<DIR>/devcontainer.json (<DIR> は 1 階層だけ。複数あれば、名前の辞書順)
//  3. .devcontainer.json
//
// 見つからなければ、空の一覧を返す (エラーにはしない)。
func Candidates(root string) ([]string, error) {
	var found []string

	base := filepath.Join(root, ".devcontainer")

	if p := filepath.Join(base, "devcontainer.json"); isFile(p) {
		found = append(found, p)
	}

	// .devcontainer が無い、またはディレクトリではないときは、読み取りに失敗するだけなので、無視する。
	// os.ReadDir は、名前の辞書順で返す。
	entries, _ := os.ReadDir(base)
	for _, e := range entries {
		if p := filepath.Join(base, e.Name(), "devcontainer.json"); isFile(p) {
			found = append(found, p)
		}
	}

	if p := filepath.Join(root, ".devcontainer.json"); isFile(p) {
		found = append(found, p)
	}
	return found, nil
}

// Find は、最も優先順位の高い devcontainer.json のパスを返す。
// 見つからなければ、ErrNotFound を (errors.Is で判定できる形で) 返す。
func Find(root string) (string, error) {
	found, err := Candidates(root)
	if err != nil {
		return "", err
	}
	if len(found) == 0 {
		return "", fmt.Errorf("%w in %s", ErrNotFound, root)
	}
	return found[0], nil
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
