package filter

import (
	"reflect"
	"testing"
)

func TestFilter(t *testing.T) {
	data := []byte(`{"name":"x","image":"ubuntu","initializeCommand":"echo hi","bogus":1,"Name":2}`)
	res, err := Filter(data, []string{"image", "initializeCommand"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := Keys(res.Used), []string{"image", "initializeCommand"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Used = %v, want %v", got, want)
	}
	if got, want := Keys(res.Ignored), []string{"name"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Ignored = %v, want %v", got, want)
	}
	// 大文字小文字の違うキー (Name) は、仕様外
	if got, want := Keys(res.Unknown), []string{"Name", "bogus"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Unknown = %v, want %v", got, want)
	}
	if string(res.Used["image"]) != `"ubuntu"` || string(res.Unknown["bogus"]) != `1` {
		t.Errorf("値が、生の JSON ではない: %s %s", res.Used["image"], res.Unknown["bogus"])
	}
}

func TestFilterSupportedKeysNotInSchema(t *testing.T) {
	// ツールが独自に処理対象にしたキーは、スキーマに無くても、存在すれば Used
	res, err := Filter([]byte(`{"x-tool":1,"name":"n"}`), []string{"x-tool"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := Keys(res.Used), []string{"x-tool"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Used = %v, want %v", got, want)
	}
	if got, want := Keys(res.Ignored), []string{"name"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Ignored = %v, want %v", got, want)
	}
	if len(res.Unknown) != 0 {
		t.Errorf("Unknown = %v", Keys(res.Unknown))
	}
}

func TestFilterSupportedKeyThatIsAbsent(t *testing.T) {
	res, err := Filter([]byte(`{"name":"n"}`), []string{"image"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Used) != 0 {
		t.Errorf("存在しないキーは、Used にならない: %v", Keys(res.Used))
	}
}

func TestFilterNilMeansAllKnownKeysAreUsed(t *testing.T) {
	res, err := Filter([]byte(`{"name":"n","image":"i","bogus":1}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := Keys(res.Used), []string{"image", "name"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Used = %v, want %v", got, want)
	}
	if len(res.Ignored) != 0 {
		t.Errorf("Ignored = %v", Keys(res.Ignored))
	}
	if got, want := Keys(res.Unknown), []string{"bogus"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Unknown = %v, want %v", got, want)
	}
}

func TestFilterEmptyMeansNothingIsUsed(t *testing.T) {
	res, err := Filter([]byte(`{"name":"n","bogus":1}`), []string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Used) != 0 {
		t.Errorf("Used = %v", Keys(res.Used))
	}
	if got, want := Keys(res.Ignored), []string{"name"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Ignored = %v, want %v", got, want)
	}
	if got, want := Keys(res.Unknown), []string{"bogus"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Unknown = %v, want %v", got, want)
	}
}

func TestFilterEmptyObject(t *testing.T) {
	res, err := Filter([]byte(`{}`), []string{"image"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Used == nil || res.Ignored == nil || res.Unknown == nil || len(res.Used)+len(res.Ignored)+len(res.Unknown) != 0 {
		t.Errorf("got %+v", res)
	}
}

func TestFilterErrors(t *testing.T) {
	for _, in := range []string{``, `[]`, `"x"`, `null`, `{"a":`} {
		if res, err := Filter([]byte(in), nil); err == nil {
			t.Errorf("%q: エラーになるはず: got %+v", in, res)
		}
	}
}

func TestFilterCopiesValues(t *testing.T) {
	data := []byte(`{"name":"abc"}`)
	res, err := Filter(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := range data {
		data[i] = 'x'
	}
	if string(res.Used["name"]) != `"abc"` {
		t.Errorf("入力を書き換えると、結果が変わった: %s", res.Used["name"])
	}
}
