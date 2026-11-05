// Package filter は、設定のトップレベルキーを、Used / Ignored / Unknown の 3 つに分類。
package filter

import (
    "encoding/json"
    "fmt"

    "github.com/imachange/devcontainer-parser/spec"
)

// Result は、分類の結果です。値は、各キーの、生の JSON。
//
//   - Used: 呼び出し側のツールが処理対象 (supportedKeys) としているキー
//   - Ignored: スキーマが定義しているが、ツールが処理対象にしていないキー
//   - Unknown: スキーマが定義していない (仕様外の) キー
type Result struct {
    Used    map[string][]byte
    Ignored map[string][]byte
    Unknown map[string][]byte
}

// Filter は、data (標準 JSON のオブジェクト) のトップレベルキーを分類。
//
// supportedKeys が nil のときは、スキーマが定義するすべてのキーを、処理対象として扱う。
// 空 (nil でない) のときは、処理対象が無いものとして扱う。
func Filter(data []byte, supportedKeys []string) (*Result, error) {
    var top map[string]json.RawMessage
    if err := json.Unmarshal(data, &top); err != nil {
        return nil, fmt.Errorf("filter: %w", err)
    }
    if top == nil {
        return nil, fmt.Errorf("filter: top-level value must be a JSON object")
    }
    return Top(top, supportedKeys), nil
}

// Top は、Filter と同じ分類を、読み込み済みのトップレベルのキーと値に対して行う。
//
// supportedKeys に含まれるキーは、スキーマに無いものでも、存在すれば Used に分類される。
// (ツールが、独自のキーを処理対象にしている場合のため)。
func Top(top map[string]json.RawMessage, supportedKeys []string) *Result {
    var supported map[string]struct{}
    if supportedKeys != nil {
        supported = make(map[string]struct{}, len(supportedKeys))
        for _, k := range supportedKeys {
            supported[k] = struct{}{}
        }
    }

    res := &Result{
        Used:    map[string][]byte{},
        Ignored: map[string][]byte{},
        Unknown: map[string][]byte{},
    }
    for key, raw := range top {
        value := append([]byte(nil), raw...)
        _, isSupported := supported[key]
        switch {
        case supportedKeys == nil && spec.IsKnownKey(key):
            res.Used[key] = value
        case isSupported:
            res.Used[key] = value
        case spec.IsKnownKey(key):
            res.Ignored[key] = value
        default:
            res.Unknown[key] = value
        }
    }
    return res
}

// Keys は、マップのキーを、辞書順で返す。
func Keys(m map[string][]byte) []string {
    return spec.SortedKeys(m)
}