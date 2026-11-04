package parser

// Options は、解析の設定。
type Options struct {
	// SupportedKeys は、ツールが処理対象 (使用) とするトップレベルのキー。
	// nil (指定なし) のときは、スキーマが定義するすべてのキーを、処理対象として扱う。
	// WithSupportedKeys() のように、キーを 1 つも渡したときは、空 (nil でない) になり、処理対象が無いものとして扱う。
	SupportedKeys []string

	// Strict が true のとき、スキーマに無い (Unknown の) キーがあると、*ValidationError を返す。
	Strict bool
}

// Option は、Options を変更する関数。
type Option func(*Options)

// WithSupportedKeys は、ツールが処理対象 (使用) とするキーの一覧を指定する。
// 渡したスライスは、コピーする。
func WithSupportedKeys(keys ...string) Option {
	return func(o *Options) {
		o.SupportedKeys = append([]string{}, keys...)
	}
}

// WithStrict は、スキーマに無い (Unknown の) キーが存在する場合に、エラーとするかを設定する。
func WithStrict(strict bool) Option {
	return func(o *Options) {
		o.Strict = strict
	}
}

func newOptions(opts []Option) Options {
	var o Options
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}
