package report

import (
	"strings"
	"testing"

	"fc3d-kill6/data"
	"fc3d-kill6/engine/position"
)

func TestExtendWithDigits(t *testing.T) {
	res := &position.Result{
		Positions: 3, KillCount: 2, Window: 100, Total: 120,
		Latest:     data.DigitDraw{Issue: "2026235", Date: "2026-09-02", Digits: []int{7, 7, 9}},
		Prediction: position.Prediction{Kills: [][]int{{1, 2}, {3, 4}, {5, 6}}, Models: []position.Model{position.ModelV9Local, position.ModelV9Local, position.ModelV9Local}},
		RecentN:    100, RecentRate: 60, AllRate: 55, BaselineAll: 51.2,
		Stats: []position.PositionStat{{Position: 1, Rate: 80, Baseline: 80, Model: position.ModelV9Local}},
		Rows:  []position.Row{{Issue: "2026235", Date: "2026-09-02", Open: "779", Kills: [][]int{{1, 2}, {3, 4}, {5, 6}}, AllOK: true}},
	}
	html := ExtendWithDigits(`<html><head><title>福彩3D 杀码 + 双色球 数据参考</title><style></style></head><body><div class="tabs"><input type="radio" name="lot" id="tab-3d" checked><input type="radio" name="lot" id="tab-ssq"><nav class="tab-bar"><label class="tab-btn" for="tab-3d"><span class="tab-ico">3D</span>福彩3D</label><label class="tab-btn" for="tab-ssq"><span class="tab-ico">球</span>双色球</label></nav><div class="tab-pane" id="pane-3d"></div><div class="tab-pane" id="pane-ssq"></div></div></body></html>`, res, nil)
	for _, want := range []string{"tab-p3", "排列3 三位杀码参考", "pane-p3", "V9局部公式", "2026235", "#tab-p3:checked~.tab-bar", "#tab-p5:checked~.tab-bar", "digit-history-toggle", "历史查询", "data-digit-history-search"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestUpdateDigitFooterDates(t *testing.T) {
	base := `<span class="foot-meta">3D 数据截止 2026-09-03 · 双色球截止 2026-09-03 · 每日开奖后自动更新</span>`
	p3 := &position.Result{Latest: data.DigitDraw{Date: "2026-09-03"}}
	p5 := &position.Result{Latest: data.DigitDraw{Date: "2026-09-02"}}
	got := updateDigitFooterDates(base, p3, p5)
	for _, want := range []string{"排列3 数据截止 2026-09-03", "排列5 数据截止 2026-09-02"} {
		if !strings.Contains(got, want) {
			t.Fatalf("footer missing %q: %s", want, got)
		}
	}
}
