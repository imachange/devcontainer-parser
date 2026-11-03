# `devcontainer-parser` 設計

## 1. 概要と目的

`devcontainer-parser` は、Dev Container 仕様（`devcontainer.json`）[^devcontainer-spec]の解析および評価を行う Go 言語用の軽量かつ堅牢な共通ライブラリ。devcontainer.jsonのパース、フィールド抽出、タイプ判定、自動探索、診断機能を提供する。

devcontainer.jsonのパース、フィールド抽出、タイプ判定、自動探索、診断機能を提供するGo言語用ライブラリ。

[^devcontainer-spec]: 以下のスキーマに対応
	- https://raw.githubusercontent.com/devcontainers/spec/refs/heads/main/schemas/devContainer.base.schema.json
	- https://raw.githubusercontent.com/microsoft/vscode/main/extensions/configuration-editing/schemas/devContainer.codespaces.schema.json
	- https://raw.githubusercontent.com/microsoft/vscode/main/extensions/configuration-editing/schemas/devContainer.vscode.schema.json

## 2. 主な機能

### 自動探索
指定ディレクトリから規定の優先順位に従い、対象ファイルを自動検出・取得。

|順位|パス|
|---|---|
|1|`.devcontainer/devcontainer.json`|
|2|`.devcontainer/<DIR>/devcontainer.json`|
|3|`.devcontainer.json`|

### パース & 正規化
* JSONC（コメント・末尾カンマ付き JSON）の標準 JSON へのサニタイズ。
* `initializeCommand` 等の「単一文字列」と「文字列配列」の双方の記法を単一型（`StringOrArray`）へ正規化。

### フィルタリング
* 呼び出し側ツールが指定した `SupportedKeys` に基づき、設定項目を `Used`（採用）、`Ignored`（無視）、`Unknown`（未定義）の 3 種類へ完全分離・抽出。

### タイプ判定
* `devcontainer.json` が Dockerfile ベース（`build`）、イメージベース（`image`）、Docker Compose ベース（`dockerComposeFile`）のいずれの構成であるかを判定。

### 診断ログ
* フィルタリング結果やパース状況から、CLI の `--verbose` モード等で表示可能な警告・情報メッセージを統一フォーマットで生成。

## 3. ディレクトリ・パッケージ構成

```text
devcontainer-parser/
├── parser.go                # メインエントリーポイント (Parse / ReadFile / LoadFromDir)
├── options.go               # オプション定義 (WithSupportedKeys, StrictMode 等)
├── result.go                # 解析結果 (Result) および診断メッセージ (Diagnostic)
├── errors.go                # 構造化エラー定義 (ParseError, NotFoundError 等)
│
├── discovery/               # 1. 設定ファイルの自動探索
│   ├── discovery.go
│   └── discovery_test.go
│
├── jsonc/                   # 2. JSONC サニタイズ
│   ├── jsonc.go             # Standardize (コメント/末尾カンマ除去)
│   └── jsonc_test.go
│
├── spec/                    # 構造体モデル & 型正規化
│   ├── config.go            # DevContainerConfig
│   ├── build.go             # BuildSpec
│   ├── types.go             # StringOrArray, PortMapping などカスタム Unmarshaler
│   └── spec_test.go
│
├── filter/                  # 3. キーの分類・抽出 (Used / Ignored / Unknown)
│   ├── filter.go
│   └── filter_test.go
│
└── evaluator/               # 4. 構成タイプの判定 (Build / Image / DockerCompose)
    ├── evaluator.go
    └── evaluator_test.go
```

## 4. データ構造と型設計

### 4.1. 多種表記の正規化型 (`spec/types.go`)

`devcontainer.json` 特有の柔軟な記法を Go 側で吸収。

```go
package spec

import (
	"encoding/json"
	"fmt"
)

// StringOrArray は "command" と ["cmd", "arg"] の双方を []string として正規化するカスタム型
type StringOrArray []string

func (s *StringOrArray) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*s = nil
		return nil
	}

	// 配列としてパースを試行
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = arr
		return nil
	}

	// 単一文字列としてパースを試行
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = StringOrArray{str}
		return nil
	}

	return fmt.Errorf("invalid string or array format: %s", string(data))
}
```

### 4.2. 構成タイプ (`evaluator/evaluator.go`)

```go
package evaluator

type ConfigType string

const (
	TypeBuild         ConfigType = "build"
	TypeImage         ConfigType = "image"
	TypeDockerCompose ConfigType = "dockerCompose"
	TypeUnknown       ConfigType = "unknown"
)
```

### 4.3. パース結果と診断ログ (`result.go`)

```go
package parser

import (
	"github.com/imachange/devcontainer-parser/evaluator"
	"github.com/imachange/devcontainer-parser/spec"
)

type DiagnosticLevel string

const (
	LevelInfo    DiagnosticLevel = "INFO"
	LevelWarning DiagnosticLevel = "WARN"
)

type Diagnostic struct {
	Level   DiagnosticLevel
	Key     string
	Message string
}

type Result struct {
	// 探索・読み込みされたファイルパス
	FilePath string

	// 正規化された全体設定
	Config *spec.DevContainerConfig

	// 構成タイプ (Build / Image / DockerCompose)
	Type evaluator.ConfigType

	// フィルタリング結果
	UsedFields    map[string][]byte
	IgnoredFields map[string][]byte
	UnknownFields map[string][]byte

	// 診断情報
	Diagnostics []Diagnostic
}
```

## 5. メイン API インターフェース設計

呼び出し側はルートパッケージ `parser` を使用。

### 5.1. エントリーポイント (`parser.go`)

```go
package parser

// Parse は渡された JSONC バイト列のパース、フィルタリング、診断を実行
func Parse(data []byte, opts ...Option) (*Result, error)

// ReadFile は指定パスの devcontainer.json をパース
func ReadFile(path string, opts ...Option) (*Result, error)

// LoadFromDir はディレクトリから devcontainer.json を自動探索してパース
func LoadFromDir(dir string, opts ...Option) (*Result, error)
```

### 5.2. オプション設定 (`options.go`)

```go
package parser

type Options struct {
	SupportedKeys []string
	Strict        bool
}

type Option func(*Options)

// WithSupportedKeys はツールが処理対象（使用）とするキー一覧を指定
func WithSupportedKeys(keys ...string) Option {
	return func(o *Options) {
		o.SupportedKeys = keys
	}
}

// WithStrict は未定義キーが存在する場合にエラーとするフラグを設定
func WithStrict(strict bool) Option {
	return func(o *Options) {
		o.Strict = strict
	}
}
```

## 6. 利用コードイメージ

`devcontainer-bootstrap` での導入例：

```go
package main

import (
	"fmt"
	"log"

	"github.com/imachange/devcontainer-parser"
)

func main() {
	// 1. カレントディレクトリからの自動探索および必要キーの指定パース
	res, err := parser.LoadFromDir(".", parser.WithSupportedKeys("build", "initializeCommand"))
	if err != nil {
		log.Fatalf("failed to load devcontainer.json: %v", err)
	}

	fmt.Printf("Loaded file: %s (Type: %s)\n", res.FilePath, res.Type)

	// 2. initializeCommand の参照
	if len(res.Config.InitializeCommand) > 0 {
		fmt.Printf("initializeCommand: %v\n", res.Config.InitializeCommand)
	}

	// 3. 診断ログ（無視されたフィールド等）の出力
	for _, diag := range res.Diagnostics {
		if diag.Level == parser.LevelWarning {
			log.Printf("[WARN] %s: %s", diag.Key, diag.Message)
		}
	}
}
```

## 7. 実装ロードマップ
### Phase 1: `jsonc` & `spec`
* `jsonc.Standardize` の移植および機能拡張。
* `StringOrArray` や `BuildSpec` の Unmarshaler 実装と単体テスト。
### Phase 2: `filter` & `evaluator`
* トップレベルキーの `Used` / `Ignored` / `Unknown` 分離ロジックの実装。
* 構成タイプの優先度判定ロジックの実装。
### Phase 3: `discovery` & `Diagnostic`
* 優先順位に基づくパス探索処理の実装。
* 診断メッセージ自動生成機能の実装。
### Phase 4: 統合 API 策定
ルートパッケージ `parser` の API 公開および結合テスト。