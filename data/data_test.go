package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCSVSkipsBadRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.csv")
	content := "issue,date,hundreds,tens,ones,number,raw\n" +
		"2026001,2026-01-01,1,2,3,123,1 2 3 0 0\n" +
		"bad-row-no-numbers\n" +
		"2026002,2026-01-02,4,5,6,456,4 5 6 0 0\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	draws, err := LoadCSV(path)
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if len(draws) != 2 {
		t.Fatalf("应跳过坏行得到 2 期, got %d", len(draws))
	}
	if draws[0].Issue != "2026001" || draws[1].B != 4 {
		t.Fatalf("解析错误: %+v", draws)
	}
}

func TestAppendCSVWritesHeaderOnNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fresh.csv")
	if _, err := AppendCSV(path, Draw{Issue: "2026001", Date: "2026-01-01", B: 1, S: 2, G: 3}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(raw), "issue,date") {
		t.Fatalf("新文件应写表头, got: %q", string(raw))
	}
	// 读回验证幂等：第一行表头被跳过，仅 1 期数据
	draws, err := LoadCSV(path)
	if err != nil || len(draws) != 1 {
		t.Fatalf("读回 = %d 期, err=%v", len(draws), err)
	}
	if draws[0].Issue != "2026001" {
		t.Fatalf("期号错误: %+v", draws[0])
	}
}

func TestLastIssue(t *testing.T) {
	if got := LastIssue(nil); got != "" {
		t.Fatalf("空数据 LastIssue = %q, want \"\"", got)
	}
	draws := []Draw{{Issue: "A"}, {Issue: "B"}}
	if got := LastIssue(draws); got != "B" {
		t.Fatalf("LastIssue = %q, want B", got)
	}
}
