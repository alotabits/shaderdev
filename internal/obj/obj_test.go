package obj

import (
	"os"
	"testing"
)

func TestDecode(t *testing.T) {
	file, err := os.Open("test.obj")
	if err != nil {
		t.Error(err)
	}

	o, err := Decode(file)
	if err != nil {
		t.Error(err)
	}

	if len(o.Pos) != 1026 {
		t.Error("expected 1026 Pos elements, got ", len(o.Pos))
	}

	if len(o.Tex) != 1088 {
		t.Error("expected 1088 Tex elements, got ", len(o.Tex))
	}

	if len(o.Nor) != 2048 {
		t.Error("expected 2048 Nor elements, got ", len(o.Nor))
	}

	if len(o.Face) != 2048 {
		t.Error("expected 2048 Face elements, got ", len(o.Face))
	}
}
