package parser

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/imachange/devcontainer-parser/discovery"
	"github.com/imachange/devcontainer-parser/evaluator"
	"github.com/imachange/devcontainer-parser/filter"
	"github.com/imachange/devcontainer-parser/jsonc"
	"github.com/imachange/devcontainer-parser/spec"
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

// ---------------------------------------------------------------------------
// 結合テスト: testdata/ のフィクスチャを使い、Parse / ReadFile / LoadFromDir を通しで検証する。

func integrationDir(name string) string {
	return filepath.Join("testdata", "integration", name)
}

// sortedKeys は、期待するキーを、辞書順にして返す (filter.Keys の結果と、比較するため)。
func sortedKeys(keys ...string) []string {
	out := append([]string(nil), keys...)
	sort.Strings(out)
	return out
}

// compositeKnownKeys は、testdata/integration/composite のうち、スキーマが定義するキー。
var compositeKnownKeys = []string{
	"$schema", "name",
	"dockerComposeFile", "service", "runServices", "workspaceFolder", "build", "image",
	"features", "forwardPorts", "portsAttributes", "otherPortsAttributes", "containerEnv", "remoteEnv", "mounts", "capAdd",
	"initializeCommand", "onCreateCommand", "postCreateCommand",
	"hostRequirements", "customizations",
}

func decodeFields(t *testing.T, fields map[string][]byte) map[string]any {
	t.Helper()
	out := make(map[string]any, len(fields))
	for k, raw := range fields {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatalf("%s の値が、JSON として不正: %v\n%s", k, err, raw)
		}
		out[k] = v
	}
	return out
}

func TestIntegrationCompositeFiltering(t *testing.T) {
	dir := integrationDir("composite")
	supported := []string{"dockerComposeFile", "service", "build", "image", "initializeCommand"}

	res, err := LoadFromDir(dir, WithSupportedKeys(supported...))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, ".devcontainer", "devcontainer.json"); res.FilePath != want {
		t.Errorf("FilePath = %q, want %q", res.FilePath, want)
	}
	if res.Type != evaluator.TypeDockerCompose {
		t.Errorf("Type = %q, want %q", res.Type, evaluator.TypeDockerCompose)
	}

	var ignored []string
	for _, k := range compositeKnownKeys {
		if !slices.Contains(supported, k) {
			ignored = append(ignored, k)
		}
	}
	ignored = sortedKeys(ignored...)

	if got, want := filter.Keys(res.UsedFields), sortedKeys(supported...); !reflect.DeepEqual(got, want) {
		t.Errorf("Used = %v, want %v", got, want)
	}
	if got := filter.Keys(res.IgnoredFields); !reflect.DeepEqual(got, ignored) {
		t.Errorf("Ignored = %v, want %v", got, ignored)
	}
	if got, want := filter.Keys(res.UnknownFields), []string{"myCustomKey"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Unknown = %v, want %v", got, want)
	}

	// 値は、コメントや末尾カンマが取り除かれた、正しい JSON になっている。
	for name, fields := range map[string]map[string][]byte{
		"Used": res.UsedFields, "Ignored": res.IgnoredFields, "Unknown": res.UnknownFields,
	} {
		for k, raw := range fields {
			if !json.Valid(raw) {
				t.Errorf("%s[%s] が、不正な JSON: %s", name, k, raw)
			}
		}
	}
	if got := string(res.UsedFields["service"]); got != `"app"` {
		t.Errorf(`Used["service"] = %s`, got)
	}

	// 診断: Ignored は INFO (キーの辞書順)、続いて Unknown が WARN。候補は 1 つなので、探索に関する診断は無い。
	if len(res.Diagnostics) != len(ignored)+1 {
		t.Fatalf("Diagnostics = %d 件, want %d: %v", len(res.Diagnostics), len(ignored)+1, res.Diagnostics)
	}
	for i, k := range ignored {
		if d := res.Diagnostics[i]; d.Level != LevelInfo || d.Key != k {
			t.Errorf("Diagnostics[%d] = %v, want INFO %s", i, d, k)
		}
	}
	if d := res.Diagnostics[len(ignored)]; d.Level != LevelWarning || d.Key != "myCustomKey" {
		t.Errorf("最後の診断 = %v, want WARN myCustomKey", d)
	}
	if w := res.Warnings(); len(w) != 1 || w[0].Key != "myCustomKey" {
		t.Errorf("Warnings = %v", w)
	}
}

func TestIntegrationCompositeWithEmptySupportedKeys(t *testing.T) {
	res, err := LoadFromDir(integrationDir("composite"), WithSupportedKeys())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.UsedFields) != 0 {
		t.Errorf("Used = %v", filter.Keys(res.UsedFields))
	}
	if got, want := filter.Keys(res.IgnoredFields), sortedKeys(compositeKnownKeys...); !reflect.DeepEqual(got, want) {
		t.Errorf("Ignored = %v, want %v", got, want)
	}
	if len(res.UnknownFields) != 1 {
		t.Errorf("Unknown = %v", filter.Keys(res.UnknownFields))
	}
}

func TestIntegrationCompositeConfig(t *testing.T) {
	path := filepath.Join(integrationDir("composite"), ".devcontainer", "devcontainer.json")

	// WithSupportedKeys なし: スキーマが定義するキーは、すべて Used。
	res, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := filter.Keys(res.UsedFields), sortedKeys(compositeKnownKeys...); !reflect.DeepEqual(got, want) {
		t.Errorf("Used = %v, want %v", got, want)
	}
	if len(res.IgnoredFields) != 0 {
		t.Errorf("Ignored = %v", filter.Keys(res.IgnoredFields))
	}
	if w := res.Warnings(); len(w) != 1 || w[0].Key != "myCustomKey" {
		t.Errorf("Warnings = %v", w)
	}

	c := res.Config
	if c == nil {
		t.Fatal("Config が nil")
	}

	if c.Schema == "" || c.Name != "composite" || c.Service != "app" || c.WorkspaceFolder != "/workspaces/composite" {
		t.Errorf("基本項目 = schema:%q name:%q service:%q workspaceFolder:%q", c.Schema, c.Name, c.Service, c.WorkspaceFolder)
	}
	if want := []string{"../compose.yml", "compose.override.yml"}; !reflect.DeepEqual([]string(c.DockerComposeFile), want) {
		t.Errorf("DockerComposeFile = %v", c.DockerComposeFile)
	}
	if want := []string{"app", "db"}; !reflect.DeepEqual(c.RunServices, want) {
		t.Errorf("RunServices = %v", c.RunServices)
	}
	if c.Image != "mcr.microsoft.com/devcontainers/base:bookworm" {
		t.Errorf("Image = %q", c.Image)
	}

	// build: 単一文字列の cacheFrom が、1 要素の配列になる。
	b := c.Build
	if b == nil {
		t.Fatal("Build が nil")
	}
	if b.Dockerfile != "Dockerfile" || b.Context != ".." || b.Target != "dev" {
		t.Errorf("Build = %+v", b)
	}
	if !reflect.DeepEqual(b.Args, map[string]string{"VARIANT": "bookworm"}) {
		t.Errorf("Build.Args = %v", b.Args)
	}
	if want := []string{"example/cache:latest"}; !reflect.DeepEqual([]string(b.CacheFrom), want) {
		t.Errorf("Build.CacheFrom = %v", b.CacheFrom)
	}
	if want := []string{"--network=host"}; !reflect.DeepEqual(b.Options, want) {
		t.Errorf("Build.Options = %v", b.Options)
	}

	if len(c.Features) != 2 {
		t.Errorf("Features = %v", c.Features)
	}

	// ポート: 整数と "host:port" 形式の文字列。
	wantPorts := []spec.PortMapping{{Port: 3000}, {Port: 5432, Host: "db", Spec: "db:5432"}}
	if !reflect.DeepEqual(c.ForwardPorts, wantPorts) {
		t.Errorf("ForwardPorts = %+v, want %+v", c.ForwardPorts, wantPorts)
	}
	if pa, ok := c.PortsAttributes["3000"]; !ok || pa.Label != "web" || pa.OnAutoForward != "notify" {
		t.Errorf("PortsAttributes = %+v", c.PortsAttributes)
	}
	if c.OtherPortsAttributes == nil || c.OtherPortsAttributes.OnAutoForward != "silent" {
		t.Errorf("OtherPortsAttributes = %+v", c.OtherPortsAttributes)
	}

	// 環境変数: remoteEnv は null (取り除く) を許す。
	if c.ContainerEnv["TZ"] != "Asia/Tokyo" {
		t.Errorf("ContainerEnv = %v", c.ContainerEnv)
	}
	if v := c.RemoteEnv["EDITOR"]; v == nil || *v != "vim" {
		t.Errorf(`RemoteEnv["EDITOR"] = %v`, v)
	}
	if v, ok := c.RemoteEnv["UNSET_ME"]; !ok || v != nil {
		t.Errorf(`RemoteEnv["UNSET_ME"] = %v (present=%v), want nil (present)`, v, ok)
	}

	// mounts: 文字列形式とオブジェクト形式。
	wantMounts := []spec.Mount{
		{Raw: "type=bind,source=/tmp,target=/mnt/tmp"},
		{Type: "volume", Source: "cache", Target: "/cache"},
	}
	if !reflect.DeepEqual(c.Mounts, wantMounts) {
		t.Errorf("Mounts = %+v, want %+v", c.Mounts, wantMounts)
	}
	if !reflect.DeepEqual(c.CapAdd, []string{"SYS_PTRACE"}) {
		t.Errorf("CapAdd = %v", c.CapAdd)
	}

	// ライフサイクルコマンド: 文字列 (シェル) / 配列 (シェルなし) / オブジェクト (並列)。
	if s := c.InitializeCommand.Single; s == nil || !s.Shell || !reflect.DeepEqual(s.Args, []string{"echo init"}) {
		t.Errorf("InitializeCommand = %+v", c.InitializeCommand)
	}
	if s := c.OnCreateCommand.Single; s == nil || s.Shell || !reflect.DeepEqual(s.Args, []string{"make", "setup"}) {
		t.Errorf("OnCreateCommand = %+v", c.OnCreateCommand)
	}
	par := c.PostCreateCommand.Parallel
	if len(par) != 2 || !par["deps"].Shell || par["lint"].Shell || !reflect.DeepEqual(par["lint"].Args, []string{"golangci-lint", "--version"}) {
		t.Errorf("PostCreateCommand = %+v", c.PostCreateCommand)
	}
	if !c.PostStartCommand.IsZero() {
		t.Errorf("PostStartCommand は、未指定のはず: %+v", c.PostStartCommand)
	}

	// hostRequirements: gpu は、オブジェクト形式。
	h := c.HostRequirements
	if h == nil || h.CPUs == nil || *h.CPUs != 4 || h.Memory != "8gb" {
		t.Fatalf("HostRequirements = %+v", h)
	}
	if h.GPU == nil || h.GPU.Mode != spec.GPUDetailed || h.GPU.Cores != 2 || h.GPU.Memory != "4gb" {
		t.Errorf("HostRequirements.GPU = %+v", h.GPU)
	}

	if _, ok := c.Customizations["vscode"]; !ok {
		t.Errorf("Customizations = %v", c.Customizations)
	}
}

func TestIntegrationConfigTypes(t *testing.T) {
	tests := []struct {
		dir  string
		file string // dir からの相対パス
		want evaluator.ConfigType
	}{
		{"composite", ".devcontainer/devcontainer.json", evaluator.TypeDockerCompose},
		{"build", ".devcontainer/devcontainer.json", evaluator.TypeBuild},
		{"legacy_build", ".devcontainer.json", evaluator.TypeBuild},
		{"image", ".devcontainer.json", evaluator.TypeImage},
		{"bare", ".devcontainer/devcontainer.json", evaluator.TypeUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			dir := integrationDir(tt.dir)
			res, err := LoadFromDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if want := filepath.Join(dir, filepath.FromSlash(tt.file)); res.FilePath != want {
				t.Errorf("FilePath = %q, want %q", res.FilePath, want)
			}
			if res.Type != tt.want {
				t.Errorf("Type = %q, want %q", res.Type, tt.want)
			}

			// ReadFile でも、同じ結果になる。
			res2, err := ReadFile(res.FilePath)
			if err != nil {
				t.Fatal(err)
			}
			if res2.Type != res.Type || res2.Config.Name != res.Config.Name {
				t.Errorf("ReadFile = %q/%q, LoadFromDir = %q/%q", res2.Type, res2.Config.Name, res.Type, res.Config.Name)
			}
		})
	}
}

func TestIntegrationMultipleCandidates(t *testing.T) {
	dir := integrationDir("multi")
	p1 := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	p2a := filepath.Join(dir, ".devcontainer", "alt", "devcontainer.json")
	p2b := filepath.Join(dir, ".devcontainer", "beta", "devcontainer.json")
	p3 := filepath.Join(dir, ".devcontainer.json")

	res, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.FilePath != p1 || res.Config.Name != "multi-1" {
		t.Errorf("FilePath = %q, name = %q, want %q", res.FilePath, res.Config.Name, p1)
	}

	var notice *Diagnostic
	for i := range res.Diagnostics {
		if res.Diagnostics[i].Key == "" {
			notice = &res.Diagnostics[i]
		}
	}
	if notice == nil {
		t.Fatalf("複数の候補を知らせる診断が無い: %v", res.Diagnostics)
	}
	if notice.Level != LevelInfo {
		t.Errorf("Level = %q, want INFO", notice.Level)
	}
	// 使うもの、ほかの候補 (優先順位の順)。
	order := []string{p1, p2a, p2b, p3}
	last := -1
	for _, p := range order {
		i := strings.Index(notice.Message, p)
		if i < 0 {
			t.Fatalf("診断に %q が無い: %q", p, notice.Message)
		}
		if i <= last {
			t.Errorf("診断の候補が、優先順位の順ではない: %q", notice.Message)
		}
		last = i
	}
}

func TestIntegrationStrict(t *testing.T) {
	// 仕様外のキーがあれば、*ValidationError。
	_, err := LoadFromDir(integrationDir("composite"), WithStrict(true))
	var ve *ValidationError
	if !errors.As(err, &ve) || !errors.Is(err, ErrValidation) {
		t.Fatalf("*ValidationError ではない: %v", err)
	}
	if !reflect.DeepEqual(ve.Keys, []string{"myCustomKey"}) {
		t.Errorf("Keys = %v", ve.Keys)
	}

	// 仕様外のキーが無いフィクスチャは、Strict でもエラーにならない。
	for _, name := range []string{"build", "legacy_build", "image", "bare"} {
		if _, err := LoadFromDir(integrationDir(name), WithStrict(true)); err != nil {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestIntegrationInvalidFixtures(t *testing.T) {
	t.Run("構文エラー", func(t *testing.T) {
		dir := integrationDir("invalid_syntax")
		_, err := LoadFromDir(dir)
		var pe *ParseError
		if !errors.As(err, &pe) || !errors.Is(err, ErrParse) {
			t.Fatalf("*ParseError ではない: %v", err)
		}
		if want := filepath.Join(dir, ".devcontainer.json"); pe.Path != want {
			t.Errorf("Path = %q, want %q", pe.Path, want)
		}
		var se *jsonc.SyntaxError
		if !errors.As(err, &se) {
			t.Fatalf("原因の *jsonc.SyntaxError をたどれない: %v", err)
		}
		if se.Line != 3 {
			t.Errorf("Line = %d, want 3", se.Line)
		}
	})

	t.Run("型の違反", func(t *testing.T) {
		dir := integrationDir("invalid_type")
		_, err := LoadFromDir(dir)
		var pe *ParseError
		if !errors.As(err, &pe) || !errors.Is(err, ErrParse) {
			t.Fatalf("*ParseError ではない: %v", err)
		}
		var te *json.UnmarshalTypeError
		if !errors.As(err, &te) {
			t.Errorf("原因の *json.UnmarshalTypeError をたどれない: %v", err)
		}
	})
}

func TestIntegrationReadFileOnDirectory(t *testing.T) {
	// 存在するが、ファイルではないパス。NotFound でも ParseError でもない、読み込みのエラーになる。
	_, err := ReadFile(integrationDir("composite"))
	if err == nil {
		t.Fatal("エラーになるはず")
	}
	var nf *NotFoundError
	var pe *ParseError
	if errors.As(err, &nf) || errors.As(err, &pe) {
		t.Errorf("読み込みのエラーのはず: %T %v", err, err)
	}
}

// testdata/jsonc の、正常系・異常系のフィクスチャを、Parse の入力にする。
func TestIntegrationJSONCFixtures(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("testdata", "jsonc", "ok", "*.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatal("正常系のフィクスチャが見つからない")
	}

	// ok/<n>.jsonc と ok/<n>.json は、同じ解析結果になる。
	for _, in := range inputs {
		name := strings.TrimSuffix(filepath.Base(in), ".jsonc")
		t.Run("ok/"+name, func(t *testing.T) {
			got, err := ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			want, err := ReadFile(strings.TrimSuffix(in, ".jsonc") + ".json")
			if err != nil {
				t.Fatal(err)
			}

			if got.Type != want.Type {
				t.Errorf("Type = %q, want %q", got.Type, want.Type)
			}
			if !reflect.DeepEqual(got.Diagnostics, want.Diagnostics) {
				t.Errorf("Diagnostics = %v, want %v", got.Diagnostics, want.Diagnostics)
			}
			// 生の JSON には、コメントの跡の空白が残るので、意味が同じかを比べる。
			for kind, pair := range map[string][2]map[string][]byte{
				"Used":    {got.UsedFields, want.UsedFields},
				"Ignored": {got.IgnoredFields, want.IgnoredFields},
				"Unknown": {got.UnknownFields, want.UnknownFields},
			} {
				if g, w := decodeFields(t, pair[0]), decodeFields(t, pair[1]); !reflect.DeepEqual(g, w) {
					t.Errorf("%s = %v, want %v", kind, g, w)
				}
			}
			gb, err := json.Marshal(got.Config)
			if err != nil {
				t.Fatal(err)
			}
			wb, err := json.Marshal(want.Config)
			if err != nil {
				t.Fatal(err)
			}
			if string(gb) != string(wb) {
				t.Errorf("Config = %s, want %s", gb, wb)
			}
		})
	}

	// 現実的な devcontainer.json は、構成タイプまで判定できる。
	res, err := ReadFile(filepath.Join("testdata", "jsonc", "ok", "devcontainer_realistic.jsonc"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Type != evaluator.TypeBuild || res.Config.Name != "my-project" {
		t.Errorf("Type = %q, Name = %q", res.Type, res.Config.Name)
	}
	if len(res.Warnings()) != 0 {
		t.Errorf("Warnings = %v", res.Warnings())
	}

	// error/ と invalid/ のフィクスチャは、どれも *ParseError になる。
	var bad []string
	for _, sub := range []string{"error", "invalid"} {
		files, err := filepath.Glob(filepath.Join("testdata", "jsonc", sub, "*.jsonc"))
		if err != nil {
			t.Fatal(err)
		}
		bad = append(bad, files...)
	}
	if len(bad) == 0 {
		t.Fatal("異常系のフィクスチャが見つからない")
	}
	for _, in := range bad {
		t.Run(filepath.ToSlash(strings.TrimPrefix(in, filepath.Join("testdata", "jsonc")+string(filepath.Separator))), func(t *testing.T) {
			_, err := ReadFile(in)
			var pe *ParseError
			if !errors.As(err, &pe) || pe.Path != in {
				t.Errorf("*ParseError (Path つき) ではない: %v", err)
			}
		})
	}
}
