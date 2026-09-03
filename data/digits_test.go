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

func TestAppendDigitCSVBatchKeepsOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p3.csv")
	if _, err := AppendDigitCSV(path, DigitDraw{Issue: "2026003", Date: "2026-01-03", Digits: []int{3, 3, 3}}); err != nil {
		t.Fatal(err)
	}
	rows := []DigitDraw{
		{Issue: "2026001", Date: "2026-01-01", Digits: []int{1, 1, 1}},
		{Issue: "2026002", Date: "2026-01-02", Digits: []int{2, 2, 2}},
		{Issue: "2026003", Date: "2026-01-03", Digits: []int{9, 9, 9}}, // duplicate keeps existing row
	}
	if n, err := AppendDigitCSVBatch(path, rows); err != nil || n != 2 {
		t.Fatalf("batch append n=%d err=%v", n, err)
	}
	got, err := LoadDigitCSV(path, 3)
	if err != nil || len(got) != 3 {
		t.Fatalf("got %d rows err=%v", len(got), err)
	}
	if got[0].Issue != "2026001" || got[1].Issue != "2026002" || got[2].Issue != "2026003" || got[2].Digits[0] != 3 {
		t.Fatalf("batch order/duplicate handling failed: %+v", got)
	}
}
