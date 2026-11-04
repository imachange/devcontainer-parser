package parser

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/imachange/devcontainer-parser/discovery"
	"github.com/imachange/devcontainer-parser/evaluator"
	"github.com/imachange/devcontainer-parser/filter"
	"github.com/imachange/devcontainer-parser/jsonc"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const sample = `// dev container
{
  "name": "x",
  "image": "ubuntu", // base image
  "initializeCommand": "echo hi",
  "bogus": 1,
}`

func TestParse(t *testing.T) {
	res, err := Parse([]byte(sample), WithSupportedKeys("image", "initializeCommand"))
	if err != nil {
		t.Fatal(err)
	}
	if res.FilePath != "" {
		t.Errorf("FilePath = %q", res.FilePath)
	}
	if got, want := filter.Keys(res.UsedFields), []string{"image", "initializeCommand"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Used = %v, want %v", got, want)
	}
	if got, want := filter.Keys(res.IgnoredFields), []string{"name"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Ignored = %v, want %v", got, want)
	}
	if got, want := filter.Keys(res.UnknownFields), []string{"bogus"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Unknown = %v, want %v", got, want)
	}
	if res.Type != evaluator.TypeImage {
		t.Errorf("Type = %q", res.Type)
	}
	c := res.Config.InitializeCommand
	if c.Single == nil || !c.Single.Shell || !reflect.DeepEqual(c.Single.Args, []string{"echo hi"}) {
		t.Errorf("initializeCommand = %+v", c)
	}
	if res.Config.Name != "x" || res.Config.Image != "ubuntu" {
		t.Errorf("Config = %+v", res.Config)
	}

	want := []Diagnostic{
		{Level: LevelInfo, Key: "name"},
		{Level: LevelWarning, Key: "bogus"},
	}
	if len(res.Diagnostics) != len(want) {
		t.Fatalf("Diagnostics = %v", res.Diagnostics)
	}
	for i, w := range want {
		if res.Diagnostics[i].Level != w.Level || res.Diagnostics[i].Key != w.Key {
			t.Errorf("Diagnostics[%d] = %v, want level=%s key=%s", i, res.Diagnostics[i], w.Level, w.Key)
		}
	}
}

func TestParseWithoutSupportedKeysUsesAllKnownKeys(t *testing.T) {
	res, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := filter.Keys(res.UsedFields), []string{"image", "initializeCommand", "name"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Used = %v, want %v", got, want)
	}
	if len(res.IgnoredFields) != 0 {
		t.Errorf("Ignored = %v", filter.Keys(res.IgnoredFields))
	}
	// 仕様外のキーの警告は、WithSupportedKeys の有無に関わらず、出る
	if w := res.Warnings(); len(w) != 1 || w[0].Key != "bogus" {
		t.Errorf("Warnings = %v", w)
	}
}

func TestParseWithEmptySupportedKeys(t *testing.T) {
	res, err := Parse([]byte(sample), WithSupportedKeys())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.UsedFields) != 0 || len(res.IgnoredFields) != 3 {
		t.Errorf("Used=%v Ignored=%v", filter.Keys(res.UsedFields), filter.Keys(res.IgnoredFields))
	}
}

func TestParseStrict(t *testing.T) {
	_, err := Parse([]byte(sample), WithStrict(true))
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("*ValidationError ではない: %v", err)
	}
	if !reflect.DeepEqual(ve.Keys, []string{"bogus"}) {
		t.Errorf("Keys = %v", ve.Keys)
	}
	if !errors.Is(err, ErrValidation) {
		t.Error("errors.Is(err, ErrValidation) = false")
	}

	// 仕様外のキーが無ければ、Strict でもエラーにならない
	if _, err := Parse([]byte(`{"name":"x"}`), WithStrict(true)); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestParseTypes(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want evaluator.ConfigType
	}{
		{"image", `{"image":"ubuntu"}`, evaluator.TypeImage},
		{"build", `{"build":{"dockerfile":"Dockerfile"}}`, evaluator.TypeBuild},
		{"旧形式の dockerFile", `{"dockerFile":"Dockerfile"}`, evaluator.TypeBuild},
		{"compose", `{"dockerComposeFile":"compose.yml","service":"app"}`, evaluator.TypeDockerCompose},
		{"compose と image", `{"dockerComposeFile":["a.yml"],"image":"ubuntu"}`, evaluator.TypeDockerCompose},
		{"build と image", `{"build":{"dockerfile":"D"},"image":"ubuntu"}`, evaluator.TypeBuild},
		{"どれも無い", `{"name":"x"}`, evaluator.TypeUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Parse([]byte(tt.src))
			if err != nil {
				t.Fatal(err)
			}
			if res.Type != tt.want {
				t.Errorf("Type = %q, want %q", res.Type, tt.want)
			}
		})
	}
}

func TestParseNormalizesStringOrArray(t *testing.T) {
	res, err := Parse([]byte(`{"dockerComposeFile":"a.yml","build":{"dockerfile":"D","cacheFrom":"img"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]string(res.Config.DockerComposeFile), []string{"a.yml"}) {
		t.Errorf("dockerComposeFile = %v", res.Config.DockerComposeFile)
	}
	if !reflect.DeepEqual([]string(res.Config.Build.CacheFrom), []string{"img"}) {
		t.Errorf("cacheFrom = %v", res.Config.Build.CacheFrom)
	}
}

func TestParseObjectFormLifecycleCommand(t *testing.T) {
	res, err := Parse([]byte(`{"initializeCommand":{"a":"echo a","b":["make","b"]}}`))
	if err != nil {
		t.Fatalf("オブジェクト形式は、仕様で許されている: %v", err)
	}
	if len(res.Config.InitializeCommand.Parallel) != 2 {
		t.Errorf("initializeCommand = %+v", res.Config.InitializeCommand)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"閉じていないコメント", `{"a": 1 /* oops`},
		{"JSON の構文エラー", `{"a": }`},
		{"トップレベルが配列", `[1, 2]`},
		{"トップレベルが null", `null`},
		{"空", ``},
		{"型の違反", `{"name": 1}`},
		{"build が文字列", `{"build": "Dockerfile"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Parse([]byte(tt.src))
			if err == nil {
				t.Fatalf("エラーになるはず: got %+v", res)
			}
			if res != nil {
				t.Errorf("エラーのときは、結果は nil のはず: %+v", res)
			}
			var pe *ParseError
			if !errors.As(err, &pe) || !errors.Is(err, ErrParse) {
				t.Errorf("*ParseError ではない: %T %v", err, err)
			}
		})
	}
}

func TestParseErrorKeepsTheCause(t *testing.T) {
	_, err := Parse([]byte("{\n  \"a\": 1\n  /* oops\n}"))
	var se *jsonc.SyntaxError
	if !errors.As(err, &se) {
		t.Fatalf("原因の *jsonc.SyntaxError をたどれない: %v", err)
	}
	if se.Line != 3 {
		t.Errorf("Line = %d", se.Line)
	}
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devcontainer.json")
	write(t, path, sample)

	res, err := ReadFile(path, WithSupportedKeys("image"))
	if err != nil {
		t.Fatal(err)
	}
	if res.FilePath != path {
		t.Errorf("FilePath = %q, want %q", res.FilePath, path)
	}
	if res.Type != evaluator.TypeImage {
		t.Errorf("Type = %q", res.Type)
	}
}

func TestReadFileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.json")
	_, err := ReadFile(path)
	var nf *NotFoundError
	if !errors.As(err, &nf) || nf.Path != path {
		t.Fatalf("*NotFoundError ではない: %v", err)
	}
	if !errors.Is(err, ErrNotFound) || !errors.Is(err, os.ErrNotExist) {
		t.Error("ErrNotFound / os.ErrNotExist と一致しない")
	}
}

func TestReadFileParseErrorHasPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devcontainer.json")
	write(t, path, `{"a": }`)
	_, err := ReadFile(path)
	var pe *ParseError
	if !errors.As(err, &pe) || pe.Path != path {
		t.Fatalf("*ParseError (Path つき) ではない: %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("Error() にパスが無い: %q", err.Error())
	}
}

func TestLoadFromDir(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".devcontainer", "devcontainer.json")
	write(t, path, sample)

	res, err := LoadFromDir(root, WithSupportedKeys("image", "initializeCommand"))
	if err != nil {
		t.Fatal(err)
	}
	if res.FilePath != path {
		t.Errorf("FilePath = %q, want %q", res.FilePath, path)
	}
	if res.Type != evaluator.TypeImage {
		t.Errorf("Type = %q", res.Type)
	}
	if got, want := filter.Keys(res.UsedFields), []string{"image", "initializeCommand"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Used = %v, want %v", got, want)
	}
	for _, d := range res.Diagnostics {
		if d.Key == "" {
			t.Errorf("候補が 1 つのときは、探索に関する診断は出ないはず: %v", d)
		}
	}
}

func TestLoadFromDirPriorityAndMultipleCandidates(t *testing.T) {
	root := t.TempDir()
	p1 := filepath.Join(root, ".devcontainer", "devcontainer.json")
	p2 := filepath.Join(root, ".devcontainer", "alt", "devcontainer.json")
	p3 := filepath.Join(root, ".devcontainer.json")
	write(t, p1, `{"image":"one"}`)
	write(t, p2, `{"image":"two"}`)
	write(t, p3, `{"image":"three"}`)

	res, err := LoadFromDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.FilePath != p1 || res.Config.Image != "one" {
		t.Errorf("FilePath = %q, image = %q", res.FilePath, res.Config.Image)
	}
	var found *Diagnostic
	for i := range res.Diagnostics {
		if res.Diagnostics[i].Key == "" {
			found = &res.Diagnostics[i]
		}
	}
	if found == nil {
		t.Fatalf("複数の候補を知らせる診断が無い: %v", res.Diagnostics)
	}
	if found.Level != LevelInfo || !strings.Contains(found.Message, p1) || !strings.Contains(found.Message, p2) || !strings.Contains(found.Message, p3) {
		t.Errorf("診断 = %v", found)
	}

	// 優先順位の高いものを消すと、次のものが使われる
	if err := os.Remove(p1); err != nil {
		t.Fatal(err)
	}
	res, err = LoadFromDir(root)
	if err != nil || res.FilePath != p2 {
		t.Fatalf("FilePath = %v, err=%v, want %q", res, err, p2)
	}
}

func TestLoadFromDirNotFound(t *testing.T) {
	root := t.TempDir()
	_, err := LoadFromDir(root)
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("*NotFoundError ではない: %v", err)
	}
	if nf.Root != root || !reflect.DeepEqual(nf.Searched, discovery.Patterns) {
		t.Errorf("Root=%q Searched=%v", nf.Root, nf.Searched)
	}
	if !errors.Is(err, ErrNotFound) || !errors.Is(err, discovery.ErrNotFound) {
		t.Error("ErrNotFound と一致しない")
	}
}

func TestLoadFromDirPropagatesParseErrors(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".devcontainer.json")
	write(t, path, `{"a": }`)
	_, err := LoadFromDir(root)
	var pe *ParseError
	if !errors.As(err, &pe) || pe.Path != path {
		t.Fatalf("*ParseError (Path つき) ではない: %v", err)
	}
}

func TestLoadFromDirStrict(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".devcontainer", "devcontainer.json"), sample)
	if _, err := LoadFromDir(root, WithStrict(true)); !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v", err)
	}
}
