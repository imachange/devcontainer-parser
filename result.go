package parser

import (
	"fmt"

	"github.com/imachange/devcontainer-parser/evaluator"
	"github.com/imachange/devcontainer-parser/filter"
	"github.com/imachange/devcontainer-parser/spec"
)

// DiagnosticLevel は、診断メッセージの重要度。
type DiagnosticLevel string

const (
	LevelInfo    DiagnosticLevel = "INFO"
	LevelWarning DiagnosticLevel = "WARN"
)

// Diagnostic は、CLI の --verbose などで表示する、警告や情報のメッセージ。
type Diagnostic struct {
	Level   DiagnosticLevel
	Key     string // 対象のキー (キーに紐づかないメッセージでは空)
	Message string
}

// String は、"[WARN] key: message" の形式で返す。
func (d Diagnostic) String() string {
	if d.Key != "" {
		return fmt.Sprintf("[%s] %s: %s", d.Level, d.Key, d.Message)
	}
	return fmt.Sprintf("[%s] %s", d.Level, d.Message)
}

// Result は、解析結果。
type Result struct {
	// 探索・読み込みされたファイルパス (Parse に、バイト列を直接渡したときは空)
	FilePath string

	// 正規化された全体設定
	Config *spec.DevContainerConfig

	// 構成タイプ (Build / Image / DockerCompose)
	Type evaluator.ConfigType

	// フィルタリング結果。値は、各キーの、生の JSON。
	UsedFields    map[string][]byte
	IgnoredFields map[string][]byte
	UnknownFields map[string][]byte

	// 診断情報
	Diagnostics []Diagnostic
}

// Warnings は、Diagnostics のうち、WARN だけを返す。
func (r *Result) Warnings() []Diagnostic {
	var out []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Level == LevelWarning {
			out = append(out, d)
		}
	}
	return out
}

// diagnose は、フィルタリング結果から、診断メッセージを作成する。
//
//   - Ignored (スキーマにあるが、ツールが使わないキー): INFO
//   - Unknown (スキーマに無いキー): WARN
//
// 順序は、INFO、WARN の順で、それぞれキーの辞書順。
func diagnose(fr *filter.Result) []Diagnostic {
	var out []Diagnostic
	for _, k := range filter.Keys(fr.Ignored) {
		out = append(out, Diagnostic{
			Level:   LevelInfo,
			Key:     k,
			Message: "ignored: defined in the schema, but not used by this tool",
		})
	}
	for _, k := range filter.Keys(fr.Unknown) {
		out = append(out, Diagnostic{
			Level:   LevelWarning,
			Key:     k,
			Message: "unknown key: not defined in the dev container schema",
		})
	}
	return out
}
