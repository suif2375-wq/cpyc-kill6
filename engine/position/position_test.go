package position

import (
	"fmt"
	"testing"

	"fc3d-kill6/data"
)

func makeDraws(n, positions int) []data.DigitDraw {
	out := make([]data.DigitDraw, 0, n)
	for i := 0; i < n; i++ {
		d := data.DigitDraw{Issue: fmt.Sprintf("2026%03d", i+1), Date: "2026-01-01", Digits: make([]int, positions)}
		for p := 0; p < positions; p++ {
			d.Digits[p] = (i + p*3) % 10
		}
		out = append(out, d)
	}
	return out
}

func TestBacktestBounds(t *testing.T) {
	for _, positions := range []int{3, 5} {
		res := Backtest(makeDraws(90, positions), 2, 40)
		if res.Positions != positions || len(res.Stats) != positions {
			t.Fatalf("positions=%d result=%+v", positions, res)
		}
		if res.AllRate < 0 || res.AllRate > 100 || res.BaselineAll <= 0 || res.BaselineAll > 100 {
			t.Fatalf("invalid rates: %+v", res)
		}
		if len(res.Prediction.Kills) != positions {
			t.Fatalf("kills len=%d want %d", len(res.Prediction.Kills), positions)
		}
		for _, ks := range res.Prediction.Kills {
			if len(ks) != 2 {
				t.Fatalf("kill len=%d", len(ks))
			}
		}
	}
}

func TestFormatPrediction(t *testing.T) {
	p := Prediction{Kills: [][]int{{1, 2}, {3, 4, 5}}}
	if got := FormatPrediction(p); got != "1,2 | 3,4,5" {
		t.Fatalf("got %q", got)
	}
}

func TestGenerateRecommendations(t *testing.T) {
	draws := makeDraws(140, 3)
	pred := Predict(draws, 2, 60)
	recs := GenerateRecommendations(draws, pred, 10)
	if len(recs) != 10 {
		t.Fatalf("recommendation count=%d want 10", len(recs))
	}
	seen := map[string]bool{}
	for i, rec := range recs {
		if rec.Rank != i+1 || seen[rec.Number] {
			t.Fatalf("duplicate or rank error: %+v", rec)
		}
		seen[rec.Number] = true
		if len(rec.Digits) != 3 || len(rec.Number) != 3 {
			t.Fatalf("invalid recommendation: %+v", rec)
		}
		for p, d := range rec.Digits {
			if contains(pred.Kills[p], d) {
				t.Fatalf("recommendation %s hits kill at pos %d", rec.Number, p)
			}
		}
	}
}

func TestGenerateRecommendationsP5(t *testing.T) {
	draws := makeDraws(150, 5)
	pred := Predict(draws, 2, 60)
	recs := GenerateRecommendations(draws, pred, 10)
	if len(recs) != 10 {
		t.Fatalf("p5 recommendation count=%d want 10", len(recs))
	}
	for _, rec := range recs {
		if len(rec.Digits) != 5 || len(rec.Number) != 5 {
			t.Fatalf("invalid p5 recommendation: %+v", rec)
		}
	}
}

func TestPredictionKillsAreDistinct(t *testing.T) {
	for _, positions := range []int{3, 5} {
		pred := Predict(makeDraws(120, positions), 2, 60)
		for p, kills := range pred.Kills {
			if len(kills) != 2 || kills[0] == kills[1] {
				t.Fatalf("positions=%d pos=%d duplicate kills: %v", positions, p, kills)
			}
		}
	}
}

func TestP5PrefixMatchesP3(t *testing.T) {
	p3Draws := makeDraws(120, 3)
	p5Draws := makeDraws(120, 5)
	p3 := Predict(p3Draws, 2, 60)
	p5 := Predict(p5Draws, 2, 60)
	for p := 0; p < 3; p++ {
		if len(p3.Kills[p]) != len(p5.Kills[p]) || p3.Kills[p][0] != p5.Kills[p][0] || p3.Kills[p][1] != p5.Kills[p][1] {
			t.Fatalf("prefix mismatch at pos %d: p3=%v p5=%v", p, p3.Kills[p], p5.Kills[p])
		}
	}
}

func TestRecommendationsAreDiversified(t *testing.T) {
	for _, positions := range []int{3, 5} {
		draws := makeDraws(150, positions)
		pred := Predict(draws, 2, 60)
		recs := GenerateRecommendations(draws, pred, 10)
		if len(recs) != 10 {
			t.Fatalf("positions=%d recommendations=%d", positions, len(recs))
		}
		minDiff := 2
		if positions == 5 {
			minDiff = 3
		}
		for i := 0; i < len(recs); i++ {
			for j := i + 1; j < len(recs); j++ {
				if digitDistance(recs[i].Digits, recs[j].Digits) < minDiff {
					t.Fatalf("positions=%d recommendations too similar: %v %v", positions, recs[i], recs[j])
				}
			}
		}
	}
}
