package jsonc

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fixtureDir(parts ...string) string {
	return filepath.Join(append([]string{"..", "testdata", "jsonc"}, parts...)...)
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func decodeAny(t *testing.T, b []byte) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("decode: %v\n%s", err, b)
	}
	return v
}

func okFixtures(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(fixtureDir("ok"), "*.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("正常系のフィクスチャが見つからない")
	}
	return files
}

// 正常系: ok/<name>.jsonc を Standardize した結果が、ok/<name>.json と同じ内容になる。
func TestStandardizeFixtures(t *testing.T) {
	for _, in := range okFixtures(t) {
		name := strings.TrimSuffix(filepath.Base(in), ".jsonc")
		t.Run(name, func(t *testing.T) {
			src := readFile(t, in)
			want := readFile(t, strings.TrimSuffix(in, ".jsonc")+".json")

			got, err := Standardize(src)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(src) {
				t.Errorf("長さが変わった: got %d, want %d", len(got), len(src))
			}
			if !json.Valid(got) {
				t.Fatalf("標準の JSON になっていない:\n%s", got)
			}
			if !reflect.DeepEqual(decodeAny(t, got), decodeAny(t, want)) {
				t.Errorf("内容が違う:\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

// CRLF の改行でも、同じ結果になる (フィクスチャを、メモリ上で CRLF に変換して確かめる)。
func TestStandardizeCRLF(t *testing.T) {
	for _, in := range okFixtures(t) {
		name := strings.TrimSuffix(filepath.Base(in), ".jsonc")
		t.Run(name, func(t *testing.T) {
			src := bytes.ReplaceAll(readFile(t, in), []byte("\r\n"), []byte("\n"))
			src = bytes.ReplaceAll(src, []byte("\n"), []byte("\r\n"))
			want := readFile(t, strings.TrimSuffix(in, ".jsonc")+".json")

			got, err := Standardize(src)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(src) {
				t.Errorf("長さが変わった: got %d, want %d", len(got), len(src))
			}
			if !reflect.DeepEqual(decodeAny(t, got), decodeAny(t, want)) {
				t.Errorf("内容が違う:\n got: %q\nwant: %s", got, want)
			}
		})
	}
}

// BOM 付きでも、同じ結果になる。
func TestStandardizeBOM(t *testing.T) {
	for _, in := range okFixtures(t) {
		name := strings.TrimSuffix(filepath.Base(in), ".jsonc")
		t.Run(name, func(t *testing.T) {
			src := append([]byte("\xef\xbb\xbf"), readFile(t, in)...)
			want := readFile(t, strings.TrimSuffix(in, ".jsonc")+".json")

			got, err := Standardize(src)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(src) {
				t.Errorf("長さが変わった: got %d, want %d", len(got), len(src))
			}
			if !reflect.DeepEqual(decodeAny(t, got), decodeAny(t, want)) {
				t.Errorf("内容が違う:\n got: %q\nwant: %s", got, want)
			}
		})
	}
}

// 異常系: 閉じていないコメントや文字列は、位置つきのエラーになる。
func TestStandardizeErrors(t *testing.T) {
	tests := []struct {
		file string
		line int
		col  int
		msg  string
	}{
		{"unterminated_block_comment.jsonc", 3, 3, "unterminated block comment"},
		{"unterminated_string.jsonc", 1, 7, "unterminated string"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			src := readFile(t, filepath.Join(fixtureDir("error"), tt.file))
			out, err := Standardize(src)
			if err == nil {
				t.Fatalf("エラーになるはず: %q", out)
			}
			var se *SyntaxError
			if !errors.As(err, &se) {
				t.Fatalf("*SyntaxError ではない: %T", err)
			}
			if se.Line != tt.line || se.Column != tt.col || se.Msg != tt.msg {
				t.Errorf("got line=%d column=%d msg=%q, want line=%d column=%d msg=%q", se.Line, se.Column, se.Msg, tt.line, tt.col, tt.msg)
			}
		})
	}
}

// JSON としての誤り (カンマの重複など) は、Standardize ではエラーにしない。後続の encoding/json が検出する。
func TestStandardizeDoesNotValidateJSON(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(fixtureDir("invalid"), "*.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("invalid のフィクスチャが見つからない")
	}
	for _, in := range files {
		t.Run(filepath.Base(in), func(t *testing.T) {
			src := readFile(t, in)
			got, err := Standardize(src)
			if err != nil {
				t.Fatalf("Standardize はエラーにしないはず: %v", err)
			}
			if json.Valid(got) {
				t.Errorf("JSON として不正なままのはず: %s", got)
			}
		})
	}
}

func TestStandardizeExactOutput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"末尾カンマ (オブジェクト)", `{"a":1,}`, `{"a":1 }`},
		{"末尾カンマ (配列)", `[1,2,]`, `[1,2 ]`},
		{"空白をはさんだ末尾カンマ", "{\"a\":1,\n}", "{\"a\":1 \n}"},
		{"行コメント", "{\"a\":1} // x\n", "{\"a\":1}     \n"},
		{"ブロックコメント", `{/*x*/"a":1}`, `{     "a":1}`},
		{"改行を含むブロックコメント", "{/*a\nb*/\"a\":1}", "{   \n   \"a\":1}"},
		{"文字列中の // は残す", `{"a":"x//y"}`, `{"a":"x//y"}`},
		{"文字列中の /* は残す", `{"a":"/*x*/"}`, `{"a":"/*x*/"}`},
		{"文字列中のカンマは残す", `{"a":",}"}`, `{"a":",}"}`},
		{"エスケープした引用符", `{"a":"\"//"}`, `{"a":"\"//"}`},
		{"文字列の末尾の円記号", `{"a":"\\"}//c`, `{"a":"\\"}   `},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Standardize([]byte(tt.in))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStandardizeDoesNotModifyInput(t *testing.T) {
	in := []byte("{ // c\n \"a\": 1, }")
	orig := append([]byte(nil), in...)
	if _, err := Standardize(in); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(in, orig) {
		t.Errorf("入力が変更された: %q", in)
	}
}
