package backtest

import (
	"math"
	"testing"

	"fc3d-kill6/data"
	"fc3d-kill6/engine/ssq"
)

// mkSSQDraws 构造 N 期确定性双色球数据（红球按 i 递增取模，蓝球 i%16）
func mkSSQDraws(n int) []data.SSQDraw {
	draws := make([]data.SSQDraw, 0, n)
	for i := 1; i <= n; i++ {
		draws = append(draws, data.SSQDraw{
			Issue: string(rune('A'+i%26)) + "000",
			Date:  "2026-01-01",
			R1:    i % 33, R2: (i + 5) % 33, R3: (i + 11) % 33,
			R4: (i + 17) % 33, R5: (i + 23) % 33, R6: (i + 29) % 33,
			Blue: i%16 + 1,
		})
	}
	return draws
}

func TestSSQRedBase(t *testing.T) {
	// C(27,6)/C(33,6) ≈ 26.7%
	if v := ssqRedBase(6); math.Abs(v-26.72) > 0.1 {
		t.Fatalf("ssqRedBase(6)=%.2f, want ≈26.7", v)
	}
	// (16-3)/16 = 81.25
	if v := ssqBlueBase(3); math.Abs(v-81.25) > 0.01 {
		t.Fatalf("ssqBlueBase(3)=%.2f, want 81.25", v)
	}
}

func TestSSQBacktest(t *testing.T) {
	draws := mkSSQDraws(300)
	res := SSQBacktest(draws, 6, 3, 50, ssq.StrategyHot)
	m := res.Meta
	if m.Total != 300 {
		t.Fatalf("Total=%d, want 300", m.Total)
	}
	if len(m.KillReds) != 6 || len(m.KillBlues) != 3 {
		t.Fatalf("杀号数量错误: %d 红 %d 蓝", len(m.KillReds), len(m.KillBlues))
	}
	if m.RedPct < 0 || m.RedPct > 100 || m.AllPct < 0 || m.AllPct > 100 {
		t.Fatalf("命中率越界: %+v", m)
	}
	if len(m.WF) == 0 {
		t.Fatalf("walk-forward 结果为空")
	}
	for _, w := range m.WF {
		if math.IsNaN(w.Z) || w.PVal < 0 || w.PVal > 1 {
			t.Fatalf("WF 非法: %+v", w)
		}
	}
}
