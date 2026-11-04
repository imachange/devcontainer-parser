package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/imachange/devcontainer-parser/discovery"
)

// 原因の種類を、errors.Is で判定するための値です。
var (
	// ErrNotFound は、devcontainer.json が見つからなかったことを表します (discovery.ErrNotFound と同じ値)。
	ErrNotFound = discovery.ErrNotFound
	// ErrParse は、devcontainer.json を読み取れなかったこと (構文や型の誤り) を表します。
	ErrParse = errors.New("failed to parse devcontainer.json")
	// ErrValidation は、devcontainer.json が、検証 (WithStrict など) に違反したことを表します。
	ErrValidation = errors.New("invalid devcontainer.json")
)

// ParseError は、devcontainer.json を読み取れなかったことを表します。
// 原因 (Err) は、*jsonc.SyntaxError、*json.SyntaxError、*json.UnmarshalTypeError などです。
type ParseError struct {
	Path string // 読み込んだファイル (Parse に、バイト列を直接渡したときは空)
	Err  error
}

func (e *ParseError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("parse %s: %v", e.Path, e.Err)
	}
	return fmt.Sprintf("parse devcontainer.json: %v", e.Err)
}

// Unwrap は、原因を返します (errors.Is / errors.As で、原因をたどれます)。
func (e *ParseError) Unwrap() error { return e.Err }

// Is は、ErrParse と一致します。
func (e *ParseError) Is(target error) bool { return target == ErrParse }

// NotFoundError は、devcontainer.json が見つからなかったことを表します。
type NotFoundError struct {
	Path     string   // ReadFile に渡されたパス (LoadFromDir のときは空)
	Root     string   // LoadFromDir で探索したディレクトリ (ReadFile のときは空)
	Searched []string // LoadFromDir で探した場所 (discovery.Patterns)
	Err      error    // 原因 (os.ErrNotExist など。無いこともある)
}

func (e *NotFoundError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("devcontainer.json not found: %s", e.Path)
	}
	return fmt.Sprintf("devcontainer.json not found in %s (searched: %s)", e.Root, strings.Join(e.Searched, ", "))
}

// Unwrap は、原因を返します。
func (e *NotFoundError) Unwrap() error { return e.Err }

// Is は、ErrNotFound と一致します。
func (e *NotFoundError) Is(target error) bool { return target == ErrNotFound }

// ValidationError は、検証に違反したことを表します。
type ValidationError struct {
	Keys    []string // 違反したキー (辞書順)
	Message string
}

func (e *ValidationError) Error() string {
	if len(e.Keys) == 0 {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Message, strings.Join(e.Keys, ", "))
}

// Is は、ErrValidation と一致します。
func (e *ValidationError) Is(target error) bool { return target == ErrValidation }
