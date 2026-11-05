package spec

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// StringOrArray

func TestStringOrArray(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    StringOrArray
		wantErr bool
	}{
		{"文字列", `"a"`, StringOrArray{"a"}, false},
		{"配列", `["a","b"]`, StringOrArray{"a", "b"}, false},
		{"空配列", `[]`, StringOrArray{}, false},
		{"null", `null`, nil, false},
		{"空文字列", `""`, StringOrArray{""}, false},
		{"空白を含む文字列", `"echo hi && echo bye"`, StringOrArray{"echo hi && echo bye"}, false},
		{"数値", `1`, nil, true},
		{"真偽値", `true`, nil, true},
		{"オブジェクト", `{"a":"b"}`, nil, true},
		{"配列に数値", `["a",1]`, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got StringOrArray
			err := json.Unmarshal([]byte(tt.in), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーになるはず: got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestStringOrArrayInStruct(t *testing.T) {
	var v struct {
		A StringOrArray `json:"a"`
		B StringOrArray `json:"b"`
		C StringOrArray `json:"c"`
	}
	if err := json.Unmarshal([]byte(`{"a":"x","b":["y","z"]}`), &v); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(v.A, StringOrArray{"x"}) || !reflect.DeepEqual(v.B, StringOrArray{"y", "z"}) || v.C != nil {
		t.Errorf("got %#v", v)
	}
}

func TestStringOrArrayEncodesAsArray(t *testing.T) {
	b, err := json.Marshal(StringOrArray{"a"})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `["a"]` {
		t.Errorf("got %s", b)
	}
}

// ---------------------------------------------------------------------------
// PortMapping / PortList

func TestPortMapping(t *testing.T) {
	tests := []struct {
		in      string
		want    PortMapping
		wantErr bool
	}{
		{`3000`, PortMapping{Port: 3000}, false},
		{`0`, PortMapping{Port: 0}, false},
		{`65535`, PortMapping{Port: 65535}, false},
		{`"db:5432"`, PortMapping{Port: 5432, Host: "db", Spec: "db:5432"}, false},
		{`"8000:8010"`, PortMapping{Port: 8010, Host: "8000", Spec: "8000:8010"}, false}, // host:port の書式にも一致する
		{`"3000"`, PortMapping{Spec: "3000"}, false},
		{`65536`, PortMapping{}, true},
		{`-1`, PortMapping{}, true},
		{`1.5`, PortMapping{}, true},
		{`true`, PortMapping{}, true},
		{`["a"]`, PortMapping{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			var got PortMapping
			err := json.Unmarshal([]byte(tt.in), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーになるはず: got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPortMappingRoundTrip(t *testing.T) {
	for _, in := range []string{`3000`, `"db:5432"`, `"8000:8010"`} {
		var p PortMapping
		if err := json.Unmarshal([]byte(in), &p); err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != in {
			t.Errorf("got %s, want %s", b, in)
		}
	}
}

func TestPortList(t *testing.T) {
	var one PortList
	if err := json.Unmarshal([]byte(`3000`), &one); err != nil || len(one) != 1 || one[0].Port != 3000 {
		t.Errorf("単一の値: got %+v, err=%v", one, err)
	}
	var str PortList
	if err := json.Unmarshal([]byte(`"8000:8010"`), &str); err != nil || len(str) != 1 || str[0].Spec != "8000:8010" {
		t.Errorf("単一の文字列: got %+v, err=%v", str, err)
	}
	var many PortList
	if err := json.Unmarshal([]byte(`[3000,"8000:8010"]`), &many); err != nil || len(many) != 2 {
		t.Errorf("配列: got %+v, err=%v", many, err)
	}
	var bad PortList
	if err := json.Unmarshal([]byte(`[true]`), &bad); err == nil {
		t.Error("配列の中の不正な値は、エラーになるはず")
	}
}

// ---------------------------------------------------------------------------
// Mount

func TestMount(t *testing.T) {
	var s Mount
	if err := json.Unmarshal([]byte(`"type=bind,source=/a,target=/b"`), &s); err != nil || s.Raw != "type=bind,source=/a,target=/b" {
		t.Errorf("文字列形式: got %+v, err=%v", s, err)
	}
	var o Mount
	if err := json.Unmarshal([]byte(`{"type":"volume","source":"v","target":"/c"}`), &o); err != nil || o.Type != "volume" || o.Source != "v" || o.Target != "/c" || o.Raw != "" {
		t.Errorf("オブジェクト形式: got %+v, err=%v", o, err)
	}
	var bad Mount
	if err := json.Unmarshal([]byte(`123`), &bad); err == nil {
		t.Error("数値は、エラーになるはず")
	}
	for _, m := range []Mount{s, o} {
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		var back Mount
		if err := json.Unmarshal(b, &back); err != nil || back != m {
			t.Errorf("往復: got %+v (%s), want %+v, err=%v", back, b, m, err)
		}
	}
}

// ---------------------------------------------------------------------------
// GPURequirement

func TestGPURequirement(t *testing.T) {
	tests := []struct {
		in      string
		want    GPURequirement
		wantErr bool
	}{
		{`true`, GPURequirement{Mode: GPURequired}, false},
		{`false`, GPURequirement{Mode: GPUNone}, false},
		{`"optional"`, GPURequirement{Mode: GPUOptional}, false},
		{`{"cores":4,"memory":"8gb"}`, GPURequirement{Mode: GPUDetailed, Cores: 4, Memory: "8gb"}, false},
		{`"required"`, GPURequirement{}, true},
		{`1`, GPURequirement{}, true},
		{`[]`, GPURequirement{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			var got GPURequirement
			err := json.Unmarshal([]byte(tt.in), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("エラーになるはず: got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			b, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			var back GPURequirement
			if err := json.Unmarshal(b, &back); err != nil || back != got {
				t.Errorf("往復: got %+v (%s), err=%v", back, b, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// LifecycleCommand

func TestLifecycleCommand(t *testing.T) {
	var s LifecycleCommand
	if err := json.Unmarshal([]byte(`"echo hi"`), &s); err != nil {
		t.Fatal(err)
	}
	if s.Single == nil || !s.Single.Shell || !reflect.DeepEqual(s.Single.Args, []string{"echo hi"}) || s.Parallel != nil {
		t.Errorf("文字列形式: got %+v", s)
	}

	var a LifecycleCommand
	if err := json.Unmarshal([]byte(`["echo","hi"]`), &a); err != nil {
		t.Fatal(err)
	}
	if a.Single == nil || a.Single.Shell || !reflect.DeepEqual(a.Single.Args, []string{"echo", "hi"}) {
		t.Errorf("配列形式: got %+v", a)
	}

	var o LifecycleCommand
	if err := json.Unmarshal([]byte(`{"a":"echo a","b":["make","b"]}`), &o); err != nil {
		t.Fatal(err)
	}
	if o.Single != nil || len(o.Parallel) != 2 {
		t.Fatalf("オブジェクト形式: got %+v", o)
	}
	if c := o.Parallel["a"]; !c.Shell || !reflect.DeepEqual(c.Args, []string{"echo a"}) {
		t.Errorf("a: got %+v", c)
	}
	if c := o.Parallel["b"]; c.Shell || !reflect.DeepEqual(c.Args, []string{"make", "b"}) {
		t.Errorf("b: got %+v", c)
	}

	for _, z := range []string{`null`, `{}`} {
		var zero LifecycleCommand
		if err := json.Unmarshal([]byte(z), &zero); err != nil || !zero.IsZero() {
			t.Errorf("%s: got %+v, err=%v", z, zero, err)
		}
	}

	for _, bad := range []string{`1`, `true`, `[1]`, `{"a":1}`, `{"a":null}`, `{"a":{"b":"c"}}`} {
		var v LifecycleCommand
		if err := json.Unmarshal([]byte(bad), &v); err == nil {
			t.Errorf("%s: エラーになるはず: got %+v", bad, v)
		}
	}
}

func TestLifecycleCommandRoundTrip(t *testing.T) {
	for _, in := range []string{`"echo hi"`, `["echo","hi"]`, `{"a":"echo a","b":["make","b"]}`} {
		var c LifecycleCommand
		if err := json.Unmarshal([]byte(in), &c); err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != in {
			t.Errorf("got %s, want %s", b, in)
		}
	}
	b, err := json.Marshal(LifecycleCommand{})
	if err != nil || string(b) != "null" {
		t.Errorf("zero: got %s, err=%v", b, err)
	}
}

// ---------------------------------------------------------------------------
// DevContainerConfig

const fullConfig = `{
"$schema":"https://example.com/schema.json",
"name":"dev",
"features":{"ghcr.io/devcontainers/features/git:1":{"version":"latest"}},
"overrideFeatureInstallOrder":["a","b"],
"secrets":{"TOKEN":{"description":"d","documentationUrl":"https://example.com"}},
"forwardPorts":[3000,"db:5432"],
"portsAttributes":{"3000":{"label":"App","onAutoForward":"notify","protocol":"http"}},
"otherPortsAttributes":{"onAutoForward":"ignore"},
"updateRemoteUserUID":true,
"containerEnv":{"A":"1"},
"containerUser":"vscode",
"mounts":["type=bind,source=/a,target=/b",{"type":"volume","source":"v","target":"/c"}],
"init":true,
"privileged":false,
"capAdd":["SYS_PTRACE"],
"securityOpt":["seccomp=unconfined"],
"remoteEnv":{"X":"y","Z":null},
"remoteUser":"vscode",
"initializeCommand":"echo init",
"onCreateCommand":["make","setup"],
"updateContentCommand":{"a":"echo a","b":["make","b"]},
"postCreateCommand":"echo pc",
"postStartCommand":["echo","ps"],
"postAttachCommand":"echo pa",
"waitFor":"postCreateCommand",
"userEnvProbe":"loginShell",
"hostRequirements":{"cpus":2,"memory":"4gb","storage":"32gb","gpu":{"cores":4,"memory":"8gb"}},
"customizations":{"vscode":{"extensions":["golang.go"]}},
"appPort":[3000,"8000:8010"],
"runArgs":["--net=host"],
"shutdownAction":"none",
"overrideCommand":false,
"workspaceFolder":"/workspace",
"workspaceMount":"source=x,target=/workspace,type=bind",
"build":{"dockerfile":"Dockerfile","context":"..","target":"dev","args":{"V":"1"},"cacheFrom":["img1","img2"],"options":["--no-cache"]},
"dockerFile":"Legacy.Dockerfile",
"context":".",
"image":"ubuntu",
"dockerComposeFile":["a.yml","b.yml"],
"service":"app",
"runServices":["app","db"],
"extensions":["golang.go"],
"settings":{"a":1},
"devPort":3333,
"codespaces":{"x":1}
}`

func decodeConfig(t *testing.T, src string) *DevContainerConfig {
	t.Helper()
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(src), &top); err != nil {
		t.Fatal(err)
	}
	cfg, err := Decode(top)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestDecodeFullConfig(t *testing.T) {
	cfg := decodeConfig(t, fullConfig)

	if cfg.Name != "dev" || cfg.Schema != "https://example.com/schema.json" {
		t.Errorf("name/schema: %+v", cfg)
	}
	if len(cfg.ForwardPorts) != 2 || cfg.ForwardPorts[0].Port != 3000 || cfg.ForwardPorts[1].Host != "db" {
		t.Errorf("forwardPorts: %+v", cfg.ForwardPorts)
	}
	if len(cfg.Mounts) != 2 || cfg.Mounts[0].Raw == "" || cfg.Mounts[1].Type != "volume" {
		t.Errorf("mounts: %+v", cfg.Mounts)
	}
	if v, ok := cfg.RemoteEnv["Z"]; !ok || v != nil {
		t.Errorf("remoteEnv.Z は、存在して nil のはず: %+v", cfg.RemoteEnv)
	}
	if cfg.InitializeCommand.Single == nil || !cfg.InitializeCommand.Single.Shell {
		t.Errorf("initializeCommand: %+v", cfg.InitializeCommand)
	}
	if cfg.OnCreateCommand.Single == nil || cfg.OnCreateCommand.Single.Shell {
		t.Errorf("onCreateCommand: %+v", cfg.OnCreateCommand)
	}
	if len(cfg.UpdateContentCommand.Parallel) != 2 {
		t.Errorf("updateContentCommand: %+v", cfg.UpdateContentCommand)
	}
	if cfg.HostRequirements == nil || cfg.HostRequirements.GPU == nil || cfg.HostRequirements.GPU.Mode != GPUDetailed {
		t.Errorf("hostRequirements: %+v", cfg.HostRequirements)
	}
	if cfg.Build == nil || cfg.Build.Dockerfile != "Dockerfile" || cfg.Build.Target != "dev" || len(cfg.Build.CacheFrom) != 2 {
		t.Errorf("build: %+v", cfg.Build)
	}
	if cfg.DockerFile != "Legacy.Dockerfile" || cfg.Context != "." {
		t.Errorf("legacy: %q %q", cfg.DockerFile, cfg.Context)
	}
	if cfg.Image != "ubuntu" || len(cfg.DockerComposeFile) != 2 || cfg.Service != "app" {
		t.Errorf("image/compose: %q %v %q", cfg.Image, cfg.DockerComposeFile, cfg.Service)
	}
	if len(cfg.AppPort) != 2 || cfg.AppPort[1].Spec != "8000:8010" {
		t.Errorf("appPort: %+v", cfg.AppPort)
	}
	if cfg.OverrideCommand == nil || *cfg.OverrideCommand {
		t.Errorf("overrideCommand: %v", cfg.OverrideCommand)
	}
	if cfg.DevPort == nil || *cfg.DevPort != 3333 {
		t.Errorf("devPort: %v", cfg.DevPort)
	}
	if string(cfg.Settings) != `{"a":1}` || string(cfg.Codespaces) != `{"x":1}` {
		t.Errorf("settings/codespaces: %s %s", cfg.Settings, cfg.Codespaces)
	}
}

func TestDecodeRoundTrip(t *testing.T) {
	cfg := decodeConfig(t, fullConfig)
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var back DevContainerConfig
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("%v\n%s", err, b)
	}
	if !reflect.DeepEqual(cfg, &back) {
		t.Errorf("往復で内容が変わった:\n got: %+v\nwant: %+v\njson: %s", &back, cfg, b)
	}
}

func TestDecodeEmpty(t *testing.T) {
	cfg := decodeConfig(t, `{}`)
	if !cfg.InitializeCommand.IsZero() || cfg.Build != nil || cfg.Image != "" {
		t.Errorf("got %+v", cfg)
	}
}

func TestDecodeTypeErrors(t *testing.T) {
	for _, src := range []string{
		`{"name": 1}`,
		`{"build": "Dockerfile"}`,
		`{"forwardPorts": [true]}`,
		`{"initializeCommand": 1}`,
		`{"hostRequirements": {"gpu": "yes"}}`,
		`{"dockerComposeFile": {"a": "b"}}`,
	} {
		var top map[string]json.RawMessage
		if err := json.Unmarshal([]byte(src), &top); err != nil {
			t.Fatal(err)
		}
		if cfg, err := Decode(top); err == nil {
			t.Errorf("%s: エラーになるはず: got %+v", src, cfg)
		}
	}
}

// encoding/json は、キーを大文字小文字を区別せずに照合するが、Decode は、仕様のキーと完全に一致するものだけを読む。
func TestDecodeIsCaseSensitiveForTopLevelKeys(t *testing.T) {
	cfg := decodeConfig(t, `{"dockerfile":"x","NAME":"n","Image":"i","dockerFile":"Right"}`)
	if cfg.Name != "" || cfg.Image != "" {
		t.Errorf("大文字小文字の違うキーが読まれた: %+v", cfg)
	}
	if cfg.DockerFile != "Right" {
		t.Errorf("dockerFile = %q, want Right", cfg.DockerFile)
	}
}

// ---------------------------------------------------------------------------
// キーの一覧

// モデル (DevContainerConfig) の json タグと、TopLevelKeys が、過不足なく一致する。
func TestModelMatchesTopLevelKeys(t *testing.T) {
	typ := reflect.TypeOf(DevContainerConfig{})
	fromModel := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		name, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
		if name == "" || name == "-" {
			t.Errorf("json タグが無いフィールド: %s", typ.Field(i).Name)
			continue
		}
		if fromModel[name] {
			t.Errorf("json タグが重複: %s", name)
		}
		fromModel[name] = true
	}
	fromCatalog := map[string]bool{}
	for _, k := range TopLevelKeys {
		if fromCatalog[k] {
			t.Errorf("TopLevelKeys が重複: %s", k)
		}
		fromCatalog[k] = true
	}
	for k := range fromCatalog {
		if !fromModel[k] {
			t.Errorf("TopLevelKeys にあるが、モデルに無い: %s", k)
		}
	}
	for k := range fromModel {
		if !fromCatalog[k] {
			t.Errorf("モデルにあるが、TopLevelKeys に無い: %s", k)
		}
	}
}

func TestTopLevelKeysAreSortedAndKnown(t *testing.T) {
	if !sort.StringsAreSorted(TopLevelKeys) {
		t.Error("TopLevelKeys が、辞書順になっていない")
	}
	for _, k := range TopLevelKeys {
		if !IsKnownKey(k) {
			t.Errorf("IsKnownKey(%q) = false", k)
		}
	}
	for _, k := range []string{"", "Name", "dockerfile", "bogus", "additionalProperties"} {
		if IsKnownKey(k) {
			t.Errorf("IsKnownKey(%q) = true", k)
		}
	}
}

func TestSortedKeys(t *testing.T) {
	got := SortedKeys(map[string][]byte{"b": nil, "a": nil, "c": nil})
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("got %v", got)
	}
	if got := SortedKeys(map[string]int(nil)); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}
