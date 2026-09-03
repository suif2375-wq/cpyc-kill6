package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDigitCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p5.csv")
	d := DigitDraw{Issue: "2026001", Date: "2026-01-01", Digits: []int{1, 2, 3, 4, 5}}
	if n, err := AppendDigitCSV(path, d); err != nil || n != 1 {
		t.Fatalf("first append n=%d err=%v", n, err)
	}
	if n, err := AppendDigitCSV(path, d); err != nil || n != 0 {
		t.Fatalf("duplicate append n=%d err=%v", n, err)
	}
	got, err := LoadDigitCSV(path, 5)
	if err != nil || len(got) != 1 {
		t.Fatalf("load len=%d err=%v", len(got), err)
	}
	if got[0].Issue != d.Issue || got[0].Digits[4] != 5 {
		t.Fatalf("unexpected draw: %+v", got[0])
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
