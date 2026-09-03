package backtest

import (
	"fmt"
	"math"
	"testing"

	"fc3d-kill6/data"
)

// mkDraws 构造 N 期确定性数据（避免依赖真实 CSV）
func mkDraws(n int) []data.Draw {
	draws := make([]data.Draw, 0, n)
	for i := 1; i <= n; i++ {
		draws = append(draws, data.Draw{
			Issue: fmt.Sprintf("2026%03d", i),
			Date:  fmt.Sprintf("2026-01-%02d", i),
			B:     i % 10, S: (i * 2) % 10, G: (i * 3) % 10,
		})
	}
	return draws
}

func TestRunAllBasic(t *testing.T) {
	draws := mkDraws(30)
	r := RunAll(draws)
	if r.Meta.Total != 30 {
		t.Fatalf("Meta.Total = %d, want 30", r.Meta.Total)
	}
	if len(r.Rows) != 29 { // total-1 = 29
		t.Fatalf("Rows 长度 = %d, want 29", len(r.Rows))
	}
	if r.Meta.BacktestN != 29 {
		t.Fatalf("BacktestN = %d, want 29", r.Meta.BacktestN)
	}
	if r.Pred.H < 0 || r.Pred.H > 9 || r.Pred.H2 < 0 || r.Pred.H2 > 9 {
		t.Fatalf("Pred 越界: %+v", r.Pred)
	}
}

func TestMultiWindow(t *testing.T) {
	draws := mkDraws(120)
	out := MultiWindow(draws, []int{50, 100})
	if len(out) != 2 {
		t.Fatalf("窗口数 = %d, want 2", len(out))
	}
	for k, ws := range out {
		if ws.Overall < 0 || ws.Overall > 100 || ws.All6Pct < 0 || ws.All6Pct > 100 {
			t.Fatalf("%s 统计越界: %+v", k, ws)
		}
	}
}

func TestWalkForward(t *testing.T) {
	draws := mkDraws(150)
	wf := WalkForward(draws, []int{50, 100})
	if len(wf) != 2 {
		t.Fatalf("窗口数 = %d, want 2", len(wf))
	}
	for _, w := range wf {
		if w.N != 50 && w.N != 100 {
			t.Fatalf("窗口期数异常: %+v", w)
		}
		if w.All6 < 0 || w.All6 > w.N {
			t.Fatalf("命中数越界: %+v", w)
		}
		if math.IsNaN(w.Z) || math.IsInf(w.Z, 0) {
			t.Fatalf("z 非法: %+v", w)
		}
		if w.PVal < 0 || w.PVal > 1 {
			t.Fatalf("p 值越界: %+v", w)
		}
	}
}
