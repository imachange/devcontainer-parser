// Package parser は、devcontainer.json を解析します。
//
// JSONC のサニタイズ、トップレベルキーの分類 (Used / Ignored / Unknown)、構成タイプの判定、
// ファイルの自動探索、診断メッセージの生成を、まとめて行います。
//
//	res, err := parser.LoadFromDir(".", parser.WithSupportedKeys("build", "initializeCommand"))
//
// 対応するスキーマは、spec パッケージのドキュメントを参照してください。
package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/imachange/devcontainer-parser/discovery"
	"github.com/imachange/devcontainer-parser/evaluator"
	"github.com/imachange/devcontainer-parser/filter"
	"github.com/imachange/devcontainer-parser/jsonc"
	"github.com/imachange/devcontainer-parser/spec"
)

// Parse は、渡された JSONC のバイト列を、解析します (サニタイズ、フィルタリング、診断)。
//
// 読み取れないときは *ParseError を、WithStrict(true) で仕様外のキーがあるときは *ValidationError を返します。
func Parse(data []byte, opts ...Option) (*Result, error) {
	return parse("", data, newOptions(opts))
}

// ReadFile は、指定されたパスの devcontainer.json を解析します。
// ファイルが無いときは、*NotFoundError (errors.Is(err, os.ErrNotExist) も成り立つ) を返します。
func ReadFile(path string, opts ...Option) (*Result, error) {
	return readFile(path, newOptions(opts))
}

// LoadFromDir は、ディレクトリから devcontainer.json を自動探索して、解析します。
//
// 探す順序は、discovery.Candidates のとおりです。複数見つかったときは、最も優先順位の高いものを使い、
// 他のものを知らせる INFO の診断を、Diagnostics に加えます。
// 1 つも見つからないときは、*NotFoundError を返します。
func LoadFromDir(dir string, opts ...Option) (*Result, error) {
	candidates, err := discovery.Candidates(dir)
	if err != nil {
		return nil, fmt.Errorf("discover devcontainer.json in %s: %w", dir, err)
	}
	if len(candidates) == 0 {
		return nil, &NotFoundError{Root: dir, Searched: discovery.Patterns, Err: discovery.ErrNotFound}
	}

	res, err := readFile(candidates[0], newOptions(opts))
	if err != nil {
		return nil, err
	}
	if len(candidates) > 1 {
		res.Diagnostics = append(res.Diagnostics, Diagnostic{
			Level: LevelInfo,
			Message: fmt.Sprintf("multiple devcontainer.json found; using %s (also found: %s)",
				candidates[0], strings.Join(candidates[1:], ", ")),
		})
	}
	return res, nil
}

func readFile(path string, o Options) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, &NotFoundError{Path: path, Err: err}
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return parse(path, data, o)
}

func parse(path string, data []byte, o Options) (*Result, error) {
	std, err := jsonc.Standardize(data)
	if err != nil {
		return nil, &ParseError{Path: path, Err: err}
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(std, &top); err != nil {
		return nil, &ParseError{Path: path, Err: err}
	}
	if top == nil {
		return nil, &ParseError{Path: path, Err: errors.New("top-level value must be a JSON object")}
	}

	cfg, err := spec.Decode(top)
	if err != nil {
		return nil, &ParseError{Path: path, Err: err}
	}

	fr := filter.FilterTop(top, o.SupportedKeys)
	if o.Strict && len(fr.Unknown) > 0 {
		return nil, &ValidationError{Keys: filter.Keys(fr.Unknown), Message: "unknown keys"}
	}

	return &Result{
		FilePath:      path,
		Config:        cfg,
		Type:          evaluator.Evaluate(cfg),
		UsedFields:    fr.Used,
		IgnoredFields: fr.Ignored,
		UnknownFields: fr.Unknown,
		Diagnostics:   diagnose(fr),
	}, nil
}
