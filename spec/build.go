package spec

// BuildSpec は、build (Docker のビルドに関するオプション) です。
//
// 仕様では、dockerfile は必須で (build.dockerfile 形式)、旧形式では、トップレベルの dockerFile を使い、
// build には target、args、cacheFrom、options だけを置きます。
type BuildSpec struct {
	// Dockerfile の場所。devcontainer.json のあるフォルダからの相対パス。
	Dockerfile string `json:"dockerfile,omitempty"`
	// ビルドコンテキストの場所。devcontainer.json のあるフォルダからの相対パス。
	Context string `json:"context,omitempty"`
	// マルチステージビルドの、ターゲットのステージ。
	Target string `json:"target,omitempty"`
	// ビルド引数。
	Args map[string]string `json:"args,omitempty"`
	// キャッシュとして使うイメージ。
	CacheFrom StringOrArray `json:"cacheFrom,omitempty"`
	// ビルドコマンドに渡す、追加の引数。
	Options []string `json:"options,omitempty"`
}
