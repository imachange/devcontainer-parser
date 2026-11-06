# API リファレンス

`devcontainer-parser` の API ドキュメントの一覧です。

> [!NOTE]
> このディレクトリの `parser.md` などのパッケージごとのファイルは、[gomarkdoc](https://github.com/princjef/gomarkdoc) が Go のドキュメントコメントから自動生成します。**直接編集しないでください**。変更するときは、ソースコードのコメントを編集し、再生成してください (後述)。
> この `README.md` だけは、手書きのファイルです。

## パッケージ一覧

多くの場合は、ルートパッケージ [`parser`](./parser.md) だけを使えば足ります。ほかのパッケージは、`parser` の内部で使われている部品で、単体でも利用できます。

| パッケージ | ドキュメント | 概要 |
|---|---|---|
| `github.com/imachange/devcontainer-parser` | [parser.md](./parser.md) | メインのエントリーポイント。`Parse` / `ReadFile` / `LoadFromDir`、オプション、解析結果、エラー型 |
| `.../discovery` | [discovery.md](./discovery.md) | ディレクトリから `devcontainer.json` を、規定の優先順位で自動探索 |
| `.../jsonc` | [jsonc.md](./jsonc.md) | JSONC (コメント・末尾カンマ付き JSON) を標準 JSON にサニタイズ |
| `.../spec` | [spec.md](./spec.md) | `devcontainer.json` のデータモデルと、記法の揺れを吸収する型 |
| `.../filter` | [filter.md](./filter.md) | トップレベルキーを `Used` / `Ignored` / `Unknown` に分類 |
| `.../evaluator` | [evaluator.md](./evaluator.md) | 構成方式 (Dockerfile / イメージ / Docker Compose) の判定 |

## 処理の流れ

`parser` は、次の順に、各パッケージを呼び出します。

```text
LoadFromDir ──▶ discovery.Candidates    devcontainer.json を探す (優先順位つき)
                      │
ReadFile ─────────────┤                  ファイルを読む
                      ▼
Parse ────────▶ jsonc.Standardize        コメント・末尾カンマを除去
                      │
                      ▼
                spec.Decode              DevContainerConfig へ変換 (記法の揺れを正規化)
                      │
                      ├──▶ filter.Top    Used / Ignored / Unknown に分類
                      └──▶ evaluator.Evaluate    構成タイプを判定
                      │
                      ▼
                    Result               診断メッセージ (Diagnostics) つきで返す
```

## 使い方

```go
package main

import (
	"errors"
	"fmt"
	"log"

	parser "github.com/imachange/devcontainer-parser"
)

func main() {
	// カレントディレクトリから devcontainer.json を探して、解析する。
	// ツールが処理対象にするキーを、WithSupportedKeys で指定する。
	res, err := parser.LoadFromDir(".",
		parser.WithSupportedKeys("build", "image", "initializeCommand"),
	)
	if err != nil {
		var nf *parser.NotFoundError
		if errors.As(err, &nf) {
			log.Fatalf("devcontainer.json が見つかりません: %v", err)
		}
		log.Fatal(err)
	}

	fmt.Println("file:", res.FilePath)
	fmt.Println("type:", res.Type) // build / image / dockerCompose / unknown

	// 診断メッセージ (--verbose などで表示する)
	for _, d := range res.Diagnostics {
		fmt.Println(d) // [WARN] myKey: unknown key: ...
	}
}
```

バイト列を直接渡すときは `parser.Parse(data, opts...)` を、パスを指定するときは `parser.ReadFile(path, opts...)` を使います。

## 主な型

### `Result` (解析結果)

| フィールド | 内容 |
|---|---|
| `FilePath` | 読み込んだファイルのパス (`Parse` にバイト列を直接渡したときは空) |
| `Config` | 正規化された全体設定 (`*spec.DevContainerConfig`) |
| `Type` | 構成タイプ (`evaluator.ConfigType`) |
| `UsedFields` | ツールが処理対象にしたキー (値は生の JSON) |
| `IgnoredFields` | スキーマにはあるが、ツールが処理対象にしていないキー |
| `UnknownFields` | スキーマに無い (仕様外の) キー |
| `Diagnostics` | 診断メッセージ (`[]Diagnostic`) |

### キーの分類

| 分類 | 意味 | 診断 |
|---|---|---|
| `Used` | `WithSupportedKeys` で指定されたキー (指定なしのときは、スキーマが定義するすべてのキー) | なし |
| `Ignored` | スキーマにはあるが、ツールが使わないキー | `INFO` |
| `Unknown` | スキーマに無いキー | `WARN` |

### 構成タイプの判定

複数の方式が同居しているときは、次の優先順位で判定します。

| 順位 | 条件 | `Type` |
|---|---|---|
| 1 | `dockerComposeFile` がある | `dockerCompose` |
| 2 | `build`、または旧形式の `dockerFile` がある | `build` |
| 3 | `image` がある | `image` |
| - | どれも無い | `unknown` |

### 探索の優先順位

`LoadFromDir` は、次の順に探し、最初に見つかったものを使います。複数あるときは、ほかの候補を知らせる `INFO` の診断が `Diagnostics` に加わります。

| 順位 | パス |
|---|---|
| 1 | `.devcontainer/devcontainer.json` |
| 2 | `.devcontainer/<DIR>/devcontainer.json` (`<DIR>` は 1 階層だけ。複数あれば、名前の辞書順) |
| 3 | `.devcontainer.json` |

## オプション

| オプション | 内容 |
|---|---|
| `WithSupportedKeys(keys ...string)` | ツールが処理対象にするトップレベルのキーを指定する。未指定のときは、スキーマが定義するすべてのキーが対象になる。引数なしで呼ぶと、処理対象が無いものとして扱う |
| `WithStrict(strict bool)` | `true` のとき、スキーマに無いキーがあると `*ValidationError` を返す |

## エラー

いずれも `errors.As` で型を、`errors.Is` で種類を判定できます。

| 型 | 種類を表す値 | 意味 |
|---|---|---|
| `*NotFoundError` | `ErrNotFound` | `devcontainer.json` が見つからない (`ReadFile` ではファイルが存在しない場合、`LoadFromDir` では探索で 1 つも見つからない場合) |
| `*ParseError` | `ErrParse` | 構文や型の誤りで読み取れない。原因 (`*jsonc.SyntaxError`、`*json.SyntaxError`、`*json.UnmarshalTypeError` など) を、`errors.As` でたどれる |
| `*ValidationError` | `ErrValidation` | `WithStrict(true)` で、スキーマに無いキーがあった |

## ドキュメントの再生成

```bash
mise run docs:gen     # docs/api/*.md を生成
mise run docs:check   # 生成結果が、コミット済みの内容と一致するかを検証 (CI と同じ)
```

Go のドキュメントコメントを変更したら、`docs:gen` を実行して、生成されたファイルを、同じコミットに含めてください。含めないと、CI の `docs:check` が失敗します。

## 対応するスキーマ

Dev Container 仕様のスキーマに対応しています。詳細は、[設計書](../design_spec.md) を参照してください。

- [devcontainers/spec: `devContainer.base.schema.json`](https://raw.githubusercontent.com/devcontainers/spec/refs/heads/main/schemas/devContainer.base.schema.json)
- [microsoft/vscode: `devContainer.codespaces.schema.json`](https://raw.githubusercontent.com/microsoft/vscode/main/extensions/configuration-editing/schemas/devContainer.codespaces.schema.json)
- [microsoft/vscode: `devContainer.vscode.schema.json`](https://raw.githubusercontent.com/microsoft/vscode/main/extensions/configuration-editing/schemas/devContainer.vscode.schema.json)