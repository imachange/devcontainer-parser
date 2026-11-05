package parser

import (
    "errors"
    "fmt"
    "os"
    "strings"
    "testing"

    "github.com/imachange/devcontainer-parser/discovery"
    "github.com/imachange/devcontainer-parser/jsonc"
)

func TestParseErrorAsAndIs(t *testing.T) {
    cause := &jsonc.SyntaxError{Offset: 3, Line: 2, Column: 1, Msg: "unterminated block comment"}
    var err = fmt.Errorf("load: %w", &ParseError{Path: "a/devcontainer.json", Err: cause})

    var pe *ParseError
    if !errors.As(err, &pe) {
        t.Fatal("errors.As で *ParseError を取り出せない")
    }
    if pe.Path != "a/devcontainer.json" {
        t.Errorf("Path = %q", pe.Path)
    }
    if !errors.Is(err, ErrParse) {
        t.Error("errors.Is(err, ErrParse) = false")
    }
    if errors.Is(err, ErrNotFound) || errors.Is(err, ErrValidation) {
        t.Error("他の種類のエラーと、一致してはいけない")
    }
    var se *jsonc.SyntaxError
    if !errors.As(err, &se) || se.Line != 2 {
        t.Errorf("原因の *jsonc.SyntaxError を取り出せない: %v", se)
    }
    if !strings.Contains(pe.Error(), "a/devcontainer.json") || !strings.Contains(pe.Error(), "unterminated block comment") {
        t.Errorf("Error() にコンテキストが無い: %q", pe.Error())
    }
}

func TestParseErrorWithoutPath(t *testing.T) {
    pe := &ParseError{Err: errors.New("boom")}
    if strings.Contains(pe.Error(), "parse :") || !strings.Contains(pe.Error(), "boom") {
        t.Errorf("Error() = %q", pe.Error())
    }
}

func TestNotFoundError(t *testing.T) {
    // ReadFile 由来
    byPath := &NotFoundError{Path: "x/devcontainer.json", Err: os.ErrNotExist}
    var err = fmt.Errorf("wrap: %w", byPath)
    var nf *NotFoundError
    if !errors.As(err, &nf) || nf.Path != "x/devcontainer.json" {
        t.Fatalf("errors.As: %v", nf)
    }
    if !errors.Is(err, ErrNotFound) {
        t.Error("errors.Is(err, ErrNotFound) = false")
    }
    if !errors.Is(err, os.ErrNotExist) {
        t.Error("原因の os.ErrNotExist をたどれない")
    }
    if !strings.Contains(byPath.Error(), "x/devcontainer.json") {
        t.Errorf("Error() = %q", byPath.Error())
    }

    // LoadFromDir 由来
    byDir := &NotFoundError{Root: "/repo", Searched: discovery.Patterns, Err: discovery.ErrNotFound}
    if !errors.Is(byDir, ErrNotFound) || !errors.Is(byDir, discovery.ErrNotFound) {
        t.Error("ErrNotFound / discovery.ErrNotFound と一致しない")
    }
    msg := byDir.Error()
    if !strings.Contains(msg, "/repo") || !strings.Contains(msg, ".devcontainer.json") {
        t.Errorf("Error() にコンテキストが無い: %q", msg)
    }

    if errors.Is(byPath, ErrParse) || errors.Is(byPath, ErrValidation) {
        t.Error("他の種類のエラーと、一致してはいけない")
    }
}

func TestValidationError(t *testing.T) {
    var err = fmt.Errorf("wrap: %w", &ValidationError{Keys: []string{"a", "b"}, Message: "unknown keys"})
    var ve *ValidationError
    if !errors.As(err, &ve) {
        t.Fatal("errors.As で *ValidationError を取り出せない")
    }
    if len(ve.Keys) != 2 || ve.Keys[0] != "a" {
        t.Errorf("Keys = %v", ve.Keys)
    }
    if !errors.Is(err, ErrValidation) {
        t.Error("errors.Is(err, ErrValidation) = false")
    }
    if errors.Is(err, ErrParse) || errors.Is(err, ErrNotFound) {
        t.Error("他の種類のエラーと、一致してはいけない")
    }
    if msg := ve.Error(); !strings.Contains(msg, "unknown keys") || !strings.Contains(msg, "a, b") {
        t.Errorf("Error() = %q", msg)
    }
    if (&ValidationError{Message: "only message"}).Error() != "only message" {
        t.Error("キーが無いときは、メッセージだけ")
    }
}

func TestErrNotFoundIsDiscoveryErrNotFound(t *testing.T) {
    if ErrNotFound != discovery.ErrNotFound {
        t.Error("ErrNotFound は、discovery.ErrNotFound と同じ値のはず")
    }
}