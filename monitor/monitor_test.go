package monitor

import (
	"path/filepath"
	"testing"
)

func TestRecordIdempotentAndTrim(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kill6.json")
	hist, err := Record(path, 81.0, "2026222", "2026-08-20")
	if err != nil || len(hist) != 1 {
		t.Fatalf("首次记录失败: %v len=%d", err, len(hist))
	}
	// 同期号覆盖
	hist, _ = Record(path, 81.5, "2026222", "2026-08-20")
	if len(hist) != 1 || hist[0].Pct != 81.5 {
		t.Fatalf("同期号应覆盖: %+v", hist)
	}
	// 新期号追加
	hist, _ = Record(path, 80.0, "2026223", "2026-08-21")
	if len(hist) != 2 {
		t.Fatalf("新期号应追加: len=%d", len(hist))
	}
}

func TestCheckAlert(t *testing.T) {
	hist := []Entry{
		{Issue: "1", Date: "2026-06-01", Pct: 85.0},
		{Issue: "2", Date: "2026-07-01", Pct: 76.0}, // 30 天前基线
		{Issue: "3", Date: "2026-07-31", Pct: 66.0}, // 当前 66% < 70%
	}
	triggered, reasons, drop := CheckAlert(66.0, hist)
	if !triggered {
		t.Fatalf("应触发预警: reasons=%v", reasons)
	}
	if drop < 8.0 {
		t.Fatalf("单月下滑应 >=8pp, got %.1f", drop)
	}
	// 正常情况不触发
	if t2, _, _ := CheckAlert(81.0, hist); t2 {
		t.Fatalf("81%% 不应触发预警")
	}
}
