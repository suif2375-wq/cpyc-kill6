package fetch

import (
	"testing"
)

func TestParseSSQLine(t *testing.T) {
	body := []byte("2026095 2026-08-18 04 06 14 21 22 33 16 06 14 33 21 22 04 0 0 0 0\n" +
		"2026096 2026-08-20 01 04 16 22 26 31 04 04 22 16 31 26 01 0 0 0 0\n")
	lt, err := parseSSQLine(body)
	if err != nil {
		t.Fatalf("parseSSQLine: %v", err)
	}
	if lt.Issue != "2026096" || lt.Date != "2026-08-20" || lt.Blue != 4 {
		t.Fatalf("结果错误: %+v", lt)
	}
	want := [6]int{1, 4, 16, 22, 26, 31}
	if lt.Reds != want {
		t.Fatalf("红球错误: %v, want %v", lt.Reds, want)
	}
}

func TestParseSSQNums(t *testing.T) {
	// 字符串格式（灰鸟 API "01"）
	reds, blue, err := parseSSQNums([]string{"01", "04", "16", "22", "26", "31", "04"})
	if err != nil {
		t.Fatalf("parseSSQNums: %v", err)
	}
	if reds[0] != 1 || blue != 4 {
		t.Fatalf("解析错误: %v %d", reds, blue)
	}
	// 非法蓝球
	if _, _, err := parseSSQNums([]string{"01", "04", "16", "22", "26", "31", "99"}); err == nil {
		t.Fatalf("蓝球 99 应报错")
	}
}
