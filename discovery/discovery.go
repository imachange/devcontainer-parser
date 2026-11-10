// Package discovery は、ディレクトリから devcontainer.json を、規定の優先順位で探索する。
package discovery

import (
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"slices"
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

	r, err := os.OpenRoot(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer func() {
		_ = r.Close()
	}()

	base := ".devcontainer"

	if isFileInRoot(r, filepath.Join(base, "devcontainer.json")) {
		found = append(found, filepath.Join(root, base, "devcontainer.json"))
	}

	if f, err := r.Open(base); err == nil {
		if entries, err := f.Readdirnames(-1); err == nil {
			slices.Sort(entries) // 名前の辞書順にソート
			for _, name := range entries {
				p := filepath.Join(base, name, "devcontainer.json")
				if isFileInRoot(r, p) {
					found = append(found, filepath.Join(root, p))
				}
			}
		}
		_ = f.Close()
	}

	if isFileInRoot(r, ".devcontainer.json") {
		found = append(found, filepath.Join(root, ".devcontainer.json"))
	}
	return found, nil
}

// CandidatesSeq は、Candidates と同様の探索を標準イテレータ (iter.Seq2) として返す。
func CandidatesSeq(root string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		candidates, err := Candidates(root)
		if err != nil {
			yield("", err)
			return
		}
		for _, c := range candidates {
			if !yield(c, nil) {
				return
			}
		}
	}
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

func isFileInRoot(r *os.Root, relPath string) bool {
	info, err := r.Stat(relPath)
	return err == nil && info.Mode().IsRegular()
}
