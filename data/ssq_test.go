package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSSQCSVSkipsBadRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ssq.csv")
	content := "issue,date,r1,r2,r3,r4,r5,r6,blue\n" +
		"2026096,2026-08-20,01,04,16,22,26,31,04\n" +
		"bad-row\n" +
		"2026095,2026-08-18,04,06,14,21,22,33,16\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	draws, err := LoadSSQCSV(path)
	if err != nil {
		t.Fatalf("LoadSSQCSV: %v", err)
	}
	if len(draws) != 2 {
		t.Fatalf("应跳过坏行得 2 期, got %d", len(draws))
	}
	if draws[0].Issue != "2026096" || draws[0].R1 != 1 || draws[0].Blue != 4 {
		t.Fatalf("解析错误: %+v", draws[0])
	}
	if draws[0].HasRed(4) != true || draws[0].HasRed(33) != false {
		t.Fatalf("HasRed 错误: %+v", draws[0])
	}
}

func TestAppendSSQCSVIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ssq.csv")
	d := SSQDraw{Issue: "2026096", Date: "2026-08-20", R1: 1, R2: 4, R3: 16, R4: 22, R5: 26, R6: 31, Blue: 4}
	if n, err := AppendSSQCSV(path, d); err != nil || n != 1 {
		t.Fatalf("首次追加 n=%d err=%v", n, err)
	}
	if n, _ := AppendSSQCSV(path, d); n != 0 {
		t.Fatalf("重复追加应返回 0, got %d", n)
	}
	draws, err := LoadSSQCSV(path)
	if err != nil || len(draws) != 1 {
		t.Fatalf("读回 %d 期 err=%v", len(draws), err)
	}
}
