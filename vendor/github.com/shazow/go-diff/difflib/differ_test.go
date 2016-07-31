package difflib

import (
	"bytes"
	"strings"
	"testing"
)

func TestDiffer(t *testing.T) {
	differ := New()

	tests := []struct {
		a, b, want string
		err        error
	}{
		{"", "", "", nil},
		{"foo", "foo\nbar", "@@ -1 +1,2 @@\n foo\n+bar\n", nil},
		{"foo\nbar", "foo", "@@ -1,2 +1 @@\n foo\n-bar\n", nil},
		{"foo\nbar", "bar", "@@ -1,2 +1 @@\n-foo\n bar\n", nil},
		{"a\nb\nc\nd\ne\n", "a\na\nb\nd\nf\nf\n", "@@ -1,6 +1,7 @@\n+a\n a\n b\n-c\n d\n-e\n+f\n+f\n \n", nil},
		{"a\nb\nc\nd\ne\n", "c\nd\ne\nf\ng\n", "@@ -1,6 +1,6 @@\n-a\n-b\n c\n d\n e\n+f\n+g\n \n", nil},
	}

	var out bytes.Buffer
	for i, test := range tests {
		out.Reset()
		err := differ.Diff(&out, strings.NewReader(test.a), strings.NewReader(test.b))
		if err != test.err {
			t.Errorf("case #%d: incorrect error, got: %q; want: %q", i, err, test.err)
		}
		if out.String() != test.want {
			t.Errorf("case #%d: incorrect output\n got: %q\nwant: %q", i, out.String(), test.want)
		}
	}
}
