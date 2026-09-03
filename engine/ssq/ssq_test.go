package ssq

import (
	"testing"

	"fc3d-kill6/data"
)

func mkDraws(n int) []data.SSQDraw {
	draws := make([]data.SSQDraw, 0, n)
	for i := 1; i <= n; i++ {
		draws = append(draws, data.SSQDraw{
			Issue: "T" + string(rune('0'+i%10)),
			Date:  "2026-01-01",
			R1:    1, R2: 2, R3: 3, R4: 4, R5: 5, R6: i%33 + 1,
			Blue: i%16 + 1,
		})
	}
	return draws
}

func TestKillRedsCountAndRange(t *testing.T) {
	draws := mkDraws(120)
	for _, s := range []Strategy{StrategyCold, StrategyHot, StrategyMiss} {
		kr := KillReds(draws, 6, 20, s)
		if len(kr) != 6 {
			t.Fatalf("%s 应杀 6 个红球, got %d", s.StrName(), len(kr))
		}
		for _, n := range kr {
			if n < 1 || n > 33 {
				t.Fatalf("%s 红球越界: %d", s.StrName(), n)
			}
		}
		kb := KillBlues(draws, 3, 20, s)
		if len(kb) != 3 {
			t.Fatalf("%s 应杀 3 个蓝球, got %d", s.StrName(), len(kb))
		}
		for _, n := range kb {
			if n < 1 || n > 16 {
				t.Fatalf("%s 蓝球越界: %d", s.StrName(), n)
			}
		}
	}
}

func TestStats(t *testing.T) {
	draws := mkDraws(60)
	hot := FreqReds(draws, 20)
	if len(hot) != 33 {
		t.Fatalf("红球频率应有 33 个, got %d", len(hot))
	}
	if hot[0].Freq < hot[len(hot)-1].Freq {
		t.Fatalf("频率榜应降序: %+v", hot[0])
	}
	miss := MissReds(draws, 20)
	if len(miss) != 33 {
		t.Fatalf("遗漏榜应有 33 个, got %d", len(miss))
	}
	sum := SumSeries(draws, 10)
	if len(sum) != 10 {
		t.Fatalf("和值序列应有 10 个, got %d", len(sum))
	}
}
