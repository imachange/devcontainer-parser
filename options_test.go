package parser

import "testing"

func TestDefaultOptions(t *testing.T) {
	o := newOptions(nil)
	if o.SupportedKeys != nil || o.Strict {
		t.Errorf("got %+v", o)
	}
}

func TestWithSupportedKeys(t *testing.T) {
	o := newOptions([]Option{WithSupportedKeys("build", "initializeCommand")})
	if len(o.SupportedKeys) != 2 || o.SupportedKeys[0] != "build" || o.SupportedKeys[1] != "initializeCommand" {
		t.Errorf("got %v", o.SupportedKeys)
	}
}

func TestWithSupportedKeysNoArguments(t *testing.T) {
	o := newOptions([]Option{WithSupportedKeys()})
	if o.SupportedKeys == nil || len(o.SupportedKeys) != 0 {
		t.Errorf("引数なしは、空で nil でないはず: %#v", o.SupportedKeys)
	}
}

func TestWithSupportedKeysCopiesTheSlice(t *testing.T) {
	keys := []string{"a"}
	o := newOptions([]Option{WithSupportedKeys(keys...)})
	keys[0] = "z"
	if o.SupportedKeys[0] != "a" {
		t.Errorf("呼び出し側のスライスの変更が、影響した: %v", o.SupportedKeys)
	}
}

func TestWithStrict(t *testing.T) {
	if o := newOptions([]Option{WithStrict(true)}); !o.Strict {
		t.Error("Strict = false")
	}
	if o := newOptions([]Option{WithStrict(true), WithStrict(false)}); o.Strict {
		t.Error("後から指定したものが、優先されるはず")
	}
}

func TestNilOptionIsIgnored(t *testing.T) {
	o := newOptions([]Option{nil, WithStrict(true)})
	if !o.Strict {
		t.Error("nil の Option を無視して、後続を適用するはず")
	}
}
