package logfmt

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_Unit_LogFmt_Values(t *testing.T) {
	for i, tt := range []struct {
		line     string
		expected []string
	}{
		{
			`foo`,
			[]string{"foo", "true"},
		},
		{
			`foo bar`,
			[]string{"foo", "true", "bar", "true"},
		},
		{
			`foo=`,
			[]string{"foo", ""},
		},
		{
			`foo=   bar   `,
			[]string{"foo", "bar"},
		},
		{
			`level=info msg="user login" user=john id=42 success=true `,
			[]string{"level", "info", "msg", "user login", "user", "john", "id", "42", "success", "true"},
		},
		{
			`level=info msg="hello\\nworld" user=john`,
			[]string{"level", "info", "msg", "hello\\\\nworld", "user", "john"},
		},
		{
			`a="escaped\"quote\nnewline" b=plain`,
			[]string{"a", "escaped\\\"quote\\nnewline", "b", "plain"},
		},
		{
			"a=1 b=\"bar\" ƒ=2h3s r=\"esc\\tmore stuff\" d x=sf   ",
			[]string{"a", "1", "b", "bar", "ƒ", "2h3s", "r", "esc\\tmore stuff", "d", "true", "x", "sf"},
		}} {
		t.Run(fmt.Sprintf("test-%d-%s", i, tt.line), func(t *testing.T) {
			var result []string
			err := Iterate([]byte(tt.line), func(k, v []byte) bool {
				result = append(result, string(k), string(v))
				return true
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func Test_Unit_LogFmt_Values_Invalid(t *testing.T) {
	for i, tt := range []string{
		`foo="bar"xx`,
	} {
		t.Run(fmt.Sprintf("test-%d-%s", i, tt), func(t *testing.T) {
			err := Iterate([]byte(tt), func(k, v []byte) bool {
				return true
			})
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
