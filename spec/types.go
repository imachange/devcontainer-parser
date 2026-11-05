package spec

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "regexp"
    "strconv"
)

func isNull(data []byte) bool {
    return string(data) == "null"
}

// ---------------------------------------------------------------------------
// StringOrArray

// StringOrArray は、"command" と ["cmd", "arg"] の双方を []string として正規化するカスタム型。
// 文字列は、1 要素の配列になる。null は、nil になる。
type StringOrArray []string

// UnmarshalJSON は、文字列または文字列の配列を読む。
func (s *StringOrArray) UnmarshalJSON(data []byte) error {
    data = bytes.TrimSpace(data)
    if len(data) == 0 || isNull(data) {
        *s = nil
        return nil
    }
    switch data[0] {
    case '[':
        var arr []string
        if err := json.Unmarshal(data, &arr); err != nil {
            return fmt.Errorf("invalid string array: %w", err)
        }
        *s = arr
        return nil
    case '"':
        var str string
        if err := json.Unmarshal(data, &str); err != nil {
            return fmt.Errorf("invalid string: %w", err)
        }
        *s = StringOrArray{str}
        return nil
    }
    return fmt.Errorf("invalid string or array format: %s", string(data))
}

// ---------------------------------------------------------------------------
// PortMapping / PortList

var hostPortRe = regexp.MustCompile(`^([a-z0-9-]+):(\d{1,5})$`)

// PortMapping は、forwardPorts / appPort の 1 要素。整数のポート番号、または文字列。
//
// 文字列の場合は、原文を Spec に保持する。
// forwardPorts の "host:port" 形式 (^([a-z0-9-]+):(\d{1,5})$) に
// 一致するときは、Host と Port も設定する。
// appPort の "8000:8010" (ホストのポート:コンテナのポート) も、この形式に一致するので、Host="8000"、Port=8010 に
// なります。appPort として扱うときは、Spec を、Docker のポート指定として解釈する。
type PortMapping struct {
    Port int    // 整数のポート番号、または "host:port" 形式のポート番号
    Host string // "host:port" 形式のホスト部分
    Spec string // 文字列形式の原文 (整数形式では空)
}

// UnmarshalJSON は、整数または文字列を読む。
func (p *PortMapping) UnmarshalJSON(data []byte) error {
    data = bytes.TrimSpace(data)
    if len(data) == 0 || isNull(data) {
        return nil
    }
    if data[0] == '"' {
        var s string
        if err := json.Unmarshal(data, &s); err != nil {
            return fmt.Errorf("invalid port: %w", err)
        }
        *p = PortMapping{Spec: s}
        if m := hostPortRe.FindStringSubmatch(s); m != nil {
            port, err := strconv.Atoi(m[2])
            if err != nil || port > 65535 {
                return fmt.Errorf("port out of range: %q", s)
            }
            p.Host, p.Port = m[1], port
        }
        return nil
    }
    var n int
    if err := json.Unmarshal(data, &n); err != nil {
        return fmt.Errorf("invalid port %s: must be an integer or a string", string(data))
    }
    if n < 0 || n > 65535 {
        return fmt.Errorf("port out of range: %d", n)
    }
    *p = PortMapping{Port: n}
    return nil
}

// MarshalJSON は、文字列形式だったものは文字列で、そうでなければ整数で出力する。
func (p PortMapping) MarshalJSON() ([]byte, error) {
    if p.Spec != "" {
        return json.Marshal(p.Spec)
    }
    return json.Marshal(p.Port)
}

// PortList は、appPort のように、単一の値または配列を取る、ポートの一覧。
type PortList []PortMapping

// UnmarshalJSON は、単一の値を、1 要素の一覧として読む。
func (l *PortList) UnmarshalJSON(data []byte) error {
    data = bytes.TrimSpace(data)
    if len(data) == 0 || isNull(data) {
        *l = nil
        return nil
    }
    if data[0] == '[' {
        var arr []PortMapping
        if err := json.Unmarshal(data, &arr); err != nil {
            return err
        }
        *l = arr
        return nil
    }
    var one PortMapping
    if err := one.UnmarshalJSON(data); err != nil {
        return err
    }
    *l = PortList{one}
    return nil
}

// ---------------------------------------------------------------------------
// Mount

// Mount は、mounts の 1 要素。オブジェクト形式、または Docker の --mount 形式の文字列。
type Mount struct {
    Type   string `json:"type,omitempty"`
    Source string `json:"source,omitempty"`
    Target string `json:"target,omitempty"`
    // Raw は、文字列形式の原文 (オブジェクト形式では空)。
    Raw string `json:"-"`
}

// UnmarshalJSON は、文字列またはオブジェクトを読む。
func (m *Mount) UnmarshalJSON(data []byte) error {
    data = bytes.TrimSpace(data)
    if len(data) == 0 || isNull(data) {
        return nil
    }
    if data[0] == '"' {
        var s string
        if err := json.Unmarshal(data, &s); err != nil {
            return fmt.Errorf("invalid mount: %w", err)
        }
        *m = Mount{Raw: s}
        return nil
    }
    type plain Mount
    var p plain
    if err := json.Unmarshal(data, &p); err != nil {
        return fmt.Errorf("invalid mount: %w", err)
    }
    *m = Mount(p)
    return nil
}

// MarshalJSON は、文字列形式だったものは文字列で、そうでなければオブジェクトで出力。
func (m Mount) MarshalJSON() ([]byte, error) {
    if m.Raw != "" {
        return json.Marshal(m.Raw)
    }
    type plain Mount
    return json.Marshal(plain(m))
}

// ---------------------------------------------------------------------------
// GPURequirement

// GPUMode は、hostRequirements.gpu の指定の種類。
type GPUMode string

const (
    // GPUNone は GPU を無効化 (false)。
    GPUNone GPUMode = "none"
    // GPURequired は GPU を必須化 (true)。
    GPURequired GPUMode = "required"
    // GPUOptional は GPU を任意指定 ("optional")。
    GPUOptional GPUMode = "optional"
    // GPUDetailed は GPU の詳細構成オブジェクト指定。
    GPUDetailed GPUMode = "detailed"
)

// GPURequirement は、hostRequirements.gpu 。真偽値、"optional"、またはオブジェクト。
type GPURequirement struct {
    Mode   GPUMode
    Cores  int    // GPUDetailed のときだけ
    Memory string // GPUDetailed のときだけ
}

type gpuDetail struct {
    Cores  int    `json:"cores,omitempty"`
    Memory string `json:"memory,omitempty"`
}

// UnmarshalJSON は、true / false / "optional" / オブジェクトを読む。
func (g *GPURequirement) UnmarshalJSON(data []byte) error {
    data = bytes.TrimSpace(data)
    if len(data) == 0 || isNull(data) {
        return nil
    }
    switch {
    case string(data) == "true":
        *g = GPURequirement{Mode: GPURequired}
    case string(data) == "false":
        *g = GPURequirement{Mode: GPUNone}
    case data[0] == '"':
        var s string
        if err := json.Unmarshal(data, &s); err != nil {
            return fmt.Errorf("invalid gpu: %w", err)
        }
        if s != "optional" {
            return fmt.Errorf("invalid gpu %q: the only allowed string is \"optional\"", s)
        }
        *g = GPURequirement{Mode: GPUOptional}
    case data[0] == '{':
        var d gpuDetail
        if err := json.Unmarshal(data, &d); err != nil {
            return fmt.Errorf("invalid gpu: %w", err)
        }
        *g = GPURequirement{Mode: GPUDetailed, Cores: d.Cores, Memory: d.Memory}
    default:
        return fmt.Errorf("invalid gpu: %s", string(data))
    }
    return nil
}

// MarshalJSON は、読み込んだ形式のとおりに出力する。
func (g GPURequirement) MarshalJSON() ([]byte, error) {
    switch g.Mode {
    case GPURequired:
        return []byte("true"), nil
    case GPUOptional:
        return json.Marshal("optional")
    case GPUDetailed:
        return json.Marshal(gpuDetail{Cores: g.Cores, Memory: g.Memory})
    }
    return []byte("false"), nil
}

// ---------------------------------------------------------------------------
// LifecycleCommand

// Command は、1 つのコマンド。
// 文字列形式は、シェルで実行する (Args は、コマンド行全体の 1 要素、Shell は true)。
// 配列形式は、シェルを介さずに実行する (Args は、コマンドと引数、Shell は false)。
type Command struct {
    Args  []string
    Shell bool
}

// LifecycleCommand は、initializeCommand などのライフサイクルコマンド。
// 仕様では、文字列、文字列の配列、またはオブジェクト (名前 → 文字列/配列。並列に実行) のいずれか。
type LifecycleCommand struct {
    // Single は、文字列形式または配列形式のとき (オブジェクト形式では nil)。
    Single *Command
    // Parallel は、オブジェクト形式のとき。キーは、コマンドの名前。
    Parallel map[string]Command
}

// IsZero は、指定が無い (または空の) ときに true を返す。
func (c LifecycleCommand) IsZero() bool {
    return c.Single == nil && len(c.Parallel) == 0
}

func parseCommand(raw json.RawMessage) (Command, error) {
    raw = bytes.TrimSpace(raw)
    if len(raw) == 0 {
        return Command{}, errors.New("empty command")
    }
    switch raw[0] {
    case '"':
        var s string
        if err := json.Unmarshal(raw, &s); err != nil {
            return Command{}, err
        }
        return Command{Args: []string{s}, Shell: true}, nil
    case '[':
        var a []string
        if err := json.Unmarshal(raw, &a); err != nil {
            return Command{}, fmt.Errorf("a command array must contain only strings: %w", err)
        }
        return Command{Args: a}, nil
    }
    return Command{}, fmt.Errorf("a command must be a string or an array of strings: %s", string(raw))
}

func marshalCommand(c Command) ([]byte, error) {
    if c.Shell && len(c.Args) == 1 {
        return json.Marshal(c.Args[0])
    }
    if c.Args == nil {
        return json.Marshal([]string{})
    }
    return json.Marshal(c.Args)
}

// UnmarshalJSON は、文字列、配列、オブジェクトのいずれかを読む。
func (c *LifecycleCommand) UnmarshalJSON(data []byte) error {
    data = bytes.TrimSpace(data)
    if len(data) == 0 || isNull(data) {
        *c = LifecycleCommand{}
        return nil
    }
    if data[0] == '{' {
        var m map[string]json.RawMessage
        if err := json.Unmarshal(data, &m); err != nil {
            return err
        }
        par := make(map[string]Command, len(m))
        for name, raw := range m {
            cmd, err := parseCommand(raw)
            if err != nil {
                return fmt.Errorf("command %q: %w", name, err)
            }
            par[name] = cmd
        }
        *c = LifecycleCommand{Parallel: par}
        return nil
    }
    cmd, err := parseCommand(data)
    if err != nil {
        return err
    }
    *c = LifecycleCommand{Single: &cmd}
    return nil
}

// MarshalJSON は、読み込んだ形式のとおりに出力する。指定が無いときは null 。
func (c LifecycleCommand) MarshalJSON() ([]byte, error) {
    switch {
    case c.Single != nil:
        return marshalCommand(*c.Single)
    case len(c.Parallel) > 0:
        m := make(map[string]json.RawMessage, len(c.Parallel))
        for name, cmd := range c.Parallel {
            b, err := marshalCommand(cmd)
            if err != nil {
                return nil, err
            }
            m[name] = b
        }
        return json.Marshal(m)
    }
    return []byte("null"), nil
}