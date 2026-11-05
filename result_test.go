package parser

import (
	"reflect"
	"testing"

	"github.com/imachange/devcontainer-parser/filter"
)

func TestDiagnose(t *testing.T) {
	fr := &filter.Result{
		Used:    map[string][]byte{"image": []byte(`"x"`)},
		Ignored: map[string][]byte{"name": []byte(`"n"`), "features": []byte(`{}`)},
		Unknown: map[string][]byte{"zzz": []byte(`1`), "bogus": []byte(`2`)},
	}
	got := diagnose(fr)

	var keys []string
	var levels []DiagnosticLevel
	for _, d := range got {
		keys = append(keys, d.Key)
		levels = append(levels, d.Level)
		if d.Message == "" {
			t.Errorf("%s: メッセージが空", d.Key)
		}
	}
	if want := []string{"features", "name", "bogus", "zzz"}; !reflect.DeepEqual(keys, want) {
		t.Errorf("keys = %v, want %v", keys, want)
	}
	if want := []DiagnosticLevel{LevelInfo, LevelInfo, LevelWarning, LevelWarning}; !reflect.DeepEqual(levels, want) {
		t.Errorf("levels = %v, want %v", levels, want)
	}
}

func TestDiagnoseNothingToReport(t *testing.T) {
	fr := &filter.Result{
		Used:    map[string][]byte{"image": nil},
		Ignored: map[string][]byte{},
		Unknown: map[string][]byte{},
	}
	if got := diagnose(fr); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestDiagnosticString(t *testing.T) {
	tests := []struct {
		d    Diagnostic
		want string
	}{
		{Diagnostic{Level: LevelWarning, Key: "bogus", Message: "unknown key"}, "[WARN] bogus: unknown key"},
		{Diagnostic{Level: LevelInfo, Key: "name", Message: "ignored"}, "[INFO] name: ignored"},
		{Diagnostic{Level: LevelInfo, Message: "multiple files"}, "[INFO] multiple files"},
	}
	for _, tt := range tests {
		if got := tt.d.String(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestLevelValues(t *testing.T) {
	if LevelInfo != "INFO" || LevelWarning != "WARN" {
		t.Errorf("LevelInfo=%q LevelWarning=%q", LevelInfo, LevelWarning)
	}
}

func TestResultWarnings(t *testing.T) {
	r := &Result{Diagnostics: []Diagnostic{
		{Level: LevelInfo, Key: "a"},
		{Level: LevelWarning, Key: "b"},
		{Level: LevelWarning, Key: "c"},
	}}
	got := r.Warnings()
	if len(got) != 2 || got[0].Key != "b" || got[1].Key != "c" {
		t.Errorf("got %v", got)
	}
	if got := (&Result{}).Warnings(); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}
