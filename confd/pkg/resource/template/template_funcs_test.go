package template

import (
	"testing"
)

func Test_TruncateAndHashName(t *testing.T) {
	str := "This is a string that should not be truncated"
	output := TruncateAndHashName(str, len(str))
	if output != str {
		t.Errorf(`TruncateAndHashName(%s, %d) = %s, want %s`, str, len(str), output, str)
	}

	str = "This is a string that should be truncated"
	expectedLen := len(str) / 2
	output = TruncateAndHashName(str, expectedLen)
	if len(output) != expectedLen {
		t.Errorf(`TruncateAndHashName(%s, %d) has length %d, want length %d`, str, expectedLen, len(output), expectedLen)
	}
}
