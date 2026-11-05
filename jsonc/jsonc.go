// Package jsonc は、JSONC (コメントと末尾カンマを許す JSON) を、標準の JSON に直す。
//
// 出力は入力と同じ長さ。コメントと末尾カンマ、BOM は、空白に置き換える (改行は残す)。
// そのため、encoding/json が報告するバイト位置は、元のファイルの位置とそのまま対応。
//
// なお、devcontainer.json のスキーマ (devContainer.base.schema.json) は、末尾カンマを許容しない。
// (allowTrailingCommas が false) が、VS Code などは許容するので、この実装も許容。
package jsonc

import "fmt"

// SyntaxError は、JSONC として解釈できない入力 (閉じていないコメントや文字列) を表す。
type SyntaxError struct {
    Offset int    // 0 始まりのバイト位置
    Line   int    // 1 始まり
    Column int    // 1 始まり (バイト単位)
    Msg    string
}

func (e *SyntaxError) Error() string {
    return fmt.Sprintf("jsonc: %s at line %d, column %d", e.Msg, e.Line, e.Column)
}

func newSyntaxError(src []byte, offset int, msg string) *SyntaxError {
    line, col := 1, 1
    for i := 0; i < offset && i < len(src); i++ {
        if src[i] == '\n' {
            line++
            col = 1
        } else {
            col++
        }
    }
    return &SyntaxError{Offset: offset, Line: line, Column: col, Msg: msg}
}

// Standardize は、コメントと末尾カンマを取り除いた、標準の JSON を返します。入力は変更しない。
//
// 文字列の中の "//"、"/*"、"," は、そのまま残しておく。
// JSON としての正しさは検査しない (閉じていないコメントと文字列だけが、エラー)。
func Standardize(data []byte) ([]byte, error) {
    out := make([]byte, len(data))
    copy(out, data)

    // BOM は、長さを保つために、空白に置き換える。
    if len(out) >= 3 && out[0] == 0xEF && out[1] == 0xBB && out[2] == 0xBF {
        out[0], out[1], out[2] = ' ', ' ', ' '
    }

    if err := blankComments(out); err != nil {
        return nil, err
    }
    blankTrailingCommas(out)
    return out, nil
}

// blankComments は、b の中のコメントを、その場で空白に置き換える。
func blankComments(b []byte) error {
    n := len(b)
    for i := 0; i < n; i++ {
        switch b[i] {
        case '"':
            start := i
            closed := false
            for i++; i < n; i++ {
                if b[i] == '\\' {
                    i++ // 次の 1 バイトは、読み飛ばす
                    continue
                }
                if b[i] == '"' {
                    closed = true
                    break
                }
            }
            if !closed {
                return newSyntaxError(b, start, "unterminated string")
            }
        case '/':
            if i+1 < n && b[i+1] == '/' {
                for ; i < n && b[i] != '\n'; i++ {
                    if b[i] != '\r' {
                        b[i] = ' '
                    }
                }
            } else if i+1 < n && b[i+1] == '*' {
                end := -1
                for j := i + 2; j+1 < n; j++ {
                    if b[j] == '*' && b[j+1] == '/' {
                        end = j + 1
                        break
                    }
                }
                if end < 0 {
                    return newSyntaxError(b, i, "unterminated block comment")
                }
                for j := i; j <= end; j++ {
                    if b[j] != '\n' && b[j] != '\r' {
                        b[j] = ' '
                    }
                }
                i = end
            }
        }
    }
    return nil
}

// blankTrailingCommas は、"}" または "]" の直前にあるカンマを、その場で空白に置き換える。
// コメントは、先に空白に置き換えてあるものとする。
func blankTrailingCommas(b []byte) {
    n := len(b)
    inString := false
    for i := 0; i < n; i++ {
        c := b[i]
        if inString {
            switch c {
            case '\\':
                i++
            case '"':
                inString = false
            }
            continue
        }
        switch c {
        case '"':
            inString = true
        case ',':
            j := i + 1
            for j < n && isSpace(b[j]) {
                j++
            }
            if j < n && (b[j] == '}' || b[j] == ']') {
                b[i] = ' '
            }
        }
    }
}

func isSpace(c byte) bool {
    return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}