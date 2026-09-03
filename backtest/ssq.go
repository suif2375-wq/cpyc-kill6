// Package backtest 双色球回测：杀中率统计 + 随机基线对照 + walk-forward 显著性验证。
// 与福彩3D 回测同构，仅号码空间不同（红 1-33 选 6、蓝 1-16 选 1）。
package backtest

import (
	"math"

	"fc3d-kill6/data"
	"fc3d-kill6/engine/ssq"
)

// SSQRow 双色球回测明细中的一行（当期杀号 + 命中标记）
type SSQRow struct {
	Issue     string
	Date      string
	Reds      [6]int
	Blue      int
	KillReds  []int // 当期杀红（用前 window 期数据算）
	KillBlues []int // 当期杀蓝
	RedOK     bool  // 杀红全避开
	BlueOK    bool  // 杀蓝避开
	AllOK     bool  // 红蓝全避开
}

// SSQMeta 双色球回测元数据（含本期杀号与全量回测杀中率）
type SSQMeta struct {
	Total         int
	LatestIssue   string
	LatestDate    string
	NextIssue     string
	Strategy      string
	Window        int
	KillReds      []int // 本期杀红
	KillBlues     []int // 本期杀蓝
	RedN, BlueN   int
	RedPct        float64  // 全量回测：杀红全避开率
	BluePct       float64  // 全量回测：杀蓝避开率
	AllPct        float64  // 全量回测：红蓝全避开率
	BaseRed       float64  // 随机基线：杀 RedN 红全避开
	BaseBlue      float64  // 随机基线：杀 BlueN 蓝避开
	BaseAll       float64  // 随机基线：红蓝全避开
	RecentRedPct  float64  // 最近 100 期：杀红全避开率
	RecentBluePct float64  // 最近 100 期：杀蓝避开率
	RecentAllPct  float64  // 最近 100 期：红蓝全避开率
	Rows          []SSQRow // 尾部 100 期明细（最新在前）
	WF            []WFWindow
}

// SSQResult 双色球回测输出
type SSQResult struct {
	Meta SSQMeta
}

// SSQBacktest 双色球杀号回测：
// 从第 window 期开始，每期用前 window 期数据按策略算杀号，对照当期开奖。
// 输出全量杀中率（红/蓝/全中）、随机基线、尾部 100 期明细与多窗口 walk-forward。
// 注意：真实回测通常显示策略与基线相当（双色球无统计预测空间），如实呈现。
func SSQBacktest(draws []data.SSQDraw, redN, blueN, window int, s ssq.Strategy) SSQResult {
	total := len(draws)
	rhit, bhit, allhit, n := 0, 0, 0, 0
	rhit2, bhit2, allhit2, n2 := 0, 0, 0, 0 // 最近 100 期
	all := make([]SSQRow, 0, total-window)
	for t := window; t < total; t++ {
		win := draws[t-window : t]
		kr := ssq.KillReds(win, redN, 0, s)
		kb := ssq.KillBlues(win, blueN, 0, s)
		d := draws[t]
		redOK := true
		for _, r := range d.Reds() {
			if containsInt(kr, r) {
				redOK = false
				break
			}
		}
		blueOK := !containsInt(kb, d.Blue)
		if redOK {
			rhit++
		}
		if blueOK {
			bhit++
		}
		if redOK && blueOK {
			allhit++
		}
		if t >= total-100 { // 最近 100 期统计
			n2++
			if redOK {
				rhit2++
			}
			if blueOK {
				bhit2++
			}
			if redOK && blueOK {
				allhit2++
			}
		}
		all = append(all, SSQRow{
			Issue: d.Issue, Date: d.Date,
			Reds: d.Reds(), Blue: d.Blue,
			KillReds: kr, KillBlues: kb,
			RedOK: redOK, BlueOK: blueOK, AllOK: redOK && blueOK,
		})
		n++
	}

	// 本期杀号（用最近 window 期）
	lastWin := draws[total-window:]
	kr := ssq.KillReds(lastWin, redN, 0, s)
	kb := ssq.KillBlues(lastWin, blueN, 0, s)

	// 尾部 100 期明细（最新在前）
	rows := all
	if len(rows) > 100 {
		rows = rows[len(rows)-100:]
	}
	rev := make([]SSQRow, len(rows))
	for i := range rows {
		rev[len(rows)-1-i] = rows[i]
	}

	meta := SSQMeta{
		Total: total, LatestIssue: draws[total-1].Issue, LatestDate: draws[total-1].Date,
		Strategy: s.StrName(), Window: window,
		KillReds: kr, KillBlues: kb, RedN: redN, BlueN: blueN,
		RedPct: pct(rhit, n), BluePct: pct(bhit, n), AllPct: pct(allhit, n),
		BaseRed: ssqRedBase(redN), BaseBlue: ssqBlueBase(blueN),
		RecentRedPct: pct(rhit2, n2), RecentBluePct: pct(bhit2, n2), RecentAllPct: pct(allhit2, n2),
		Rows: rev,
		WF:   ssqWalkForward(draws, redN, blueN, window, s, []int{100, 200, 500}),
	}
	meta.BaseAll = meta.BaseRed * meta.BaseBlue / 100
	return SSQResult{Meta: meta}
}

// ssqRedBase 杀 n 红全避开的随机基线：C(33-n,6)/C(33,6)
func ssqRedBase(n int) float64 { return combF(33-n, 6) / combF(33, 6) * 100 }

// ssqBlueBase 杀 n 蓝避开的随机基线：(16-n)/16
func ssqBlueBase(n int) float64 { return float64(16-n) / 16 * 100 }

// ssqWalkForward 多窗口 walk-forward：每窗口独立滚动回测全中率 vs 基线，二项检验显著性。
func ssqWalkForward(draws []data.SSQDraw, redN, blueN, window int, s ssq.Strategy, windows []int) []WFWindow {
	total := len(draws)
	base := ssqRedBase(redN) * ssqBlueBase(blueN) / 100 / 100 // 全中基线（小数）
	out := []WFWindow{}
	for _, w := range windows {
		if total <= w+window {
			continue
		}
		start := total - w
		allhit, n := 0, 0
		for t := start; t < total; t++ {
			if t < window {
				continue
			}
			win := draws[t-window : t]
			kr := ssq.KillReds(win, redN, 0, s)
			kb := ssq.KillBlues(win, blueN, 0, s)
			d := draws[t]
			redOK, blueOK := true, true
			for _, r := range d.Reds() {
				if containsInt(kr, r) {
					redOK = false
					break
				}
			}
			if containsInt(kb, d.Blue) {
				blueOK = false
			}
			if redOK && blueOK {
				allhit++
			}
			n++
		}
		if n == 0 {
			continue
		}
		pctVal := pct(allhit, n)
		se := math.Sqrt(base * (1 - base) / float64(n))
		z := 0.0
		if se > 0 {
			z = (pctVal/100 - base) / se
		}
		out = append(out, WFWindow{
			Label: itoa(w) + "期", N: n, All6: allhit, All6Pct: pctVal,
			BeatPP: pctVal - base*100, Z: z, PVal: normP(z),
		})
	}
	return out
}

func containsInt(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

// combF 组合数 C(n,k)（浮点，避免中间溢出）
func combF(n, k int) float64 {
	if k > n || n < 0 {
		return 0
	}
	r := 1.0
	for i := 0; i < k; i++ {
		r *= float64(n - i)
		r /= float64(i + 1)
	}
	return r
}
