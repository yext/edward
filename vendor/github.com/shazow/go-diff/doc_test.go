package diff

import (
	"bytes"
	"fmt"
)

func ExampleDiff() {
	a := Object{
		ReadSeeker: bytes.NewReader([]byte("foo")),
		ID:         [20]byte{1}, // Fake object ID
		Path:       "myfile",
		Mode:       100644,
	}
	b := Object{
		ReadSeeker: bytes.NewReader([]byte("foo\nbar")),
		ID:         [20]byte{2}, // Another fake object ID, but it changed!
		Path:       "myfile",
		Mode:       100644,
	}

	differ := DefaultDiffer()
	out := bytes.Buffer{}
	w := Writer{
		Writer:    &out,
		Differ:    differ,
		SrcPrefix: "a/",
		DstPrefix: "b/",
	}
	w.Diff(a, b)

	fmt.Print(out.String())
	// Output:
	// diff --git a/myfile b/myfile
	// index 0100000000000000000000000000000000000000..0200000000000000000000000000000000000000 100644
	// --- a/myfile
	// +++ b/myfile
	// @@ -1 +1,2 @@
	//  foo
	// +bar
}
