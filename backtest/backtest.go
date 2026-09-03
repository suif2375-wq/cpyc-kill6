// Package backtest 提供多窗口回测、100 期明细与预测。
package backtest

import (
	"math"

	"fc3d-kill6/data"
	"fc3d-kill6/engine"
)

// Row 100 期明细中的一行（6 杀码 + 命中标记）
type Row struct {
	Issue, Date, Open string
	HK, TK, OK        int
	HK2, TK2, OK2     int
	HOK, TOK, OOK     bool
	H2OK, T2OK, O2OK  bool
	AllOK, All6OK     bool
}

// Meta 回测元数据（回测元数据）
type Meta struct {
	Total       int
	LatestIssue string
	LatestDate  string
	BacktestN   int

	AccH, AccT, AccO float64
	ErrH, ErrT, ErrO int
	AccAll           float64

	AccH2, AccT2, AccO2 float64
	ErrH2, ErrT2, ErrO2 int
	All6                int
	All6Pct             float64

	PeriodCorrect100 int
	PeriodN100       int
	AccPeriod100     float64

	Period6Correct100 int
	Period6Pct100     float64
}

// Predict 下一期预测（6 杀码）
type Predict struct {
	H, T, O    int
	H2, T2, O2 int
}

// Result RunAll 的完整输出
type Result struct {
	Meta Meta
	Rows []Row // 尾部 BACKTEST_N 期，按时间正序（与 Python results.reverse() 后一致）
	Pred Predict
}

const backtestN = 100

// RunAll 全量状态机累积回测：
// 从第 1 期开始累积 phk/ptk/pok 与个位失败追踪，只记录尾部 100 期明细；
// 明细按时间正序（最新在末尾），与 Python results.reverse() 后一致。
func RunAll(draws []data.Draw) *Result {
	total := len(draws)
	st := engine.NewState()
	all := make([]Row, 0, total-1)
	for i := 1; i < total; i++ {
		p := draws[i-1]
		d := draws[i]
		hk, tk, ok, hk2, tk2, ok2 := st.Next(p.B, p.S, p.G, d.G)
		all = append(all, Row{
			Issue: d.Issue, Date: d.Date, Open: itoa3(d.B, d.S, d.G),
			HK: hk, TK: tk, OK: ok, HK2: hk2, TK2: tk2, OK2: ok2,
			HOK: hk != d.B, TOK: tk != d.S, OOK: ok != d.G,
			H2OK: hk2 != d.B, T2OK: tk2 != d.S, O2OK: ok2 != d.G,
			AllOK:  hk != d.B && tk != d.S && ok != d.G,
			All6OK: hk != d.B && tk != d.S && ok != d.G && hk2 != d.B && tk2 != d.S && ok2 != d.G,
		})
	}

	// 尾部窗口
	n := backtestN
	if total-1 < n {
		n = total - 1
	}
	rows := all[len(all)-n:]
	rows = append([]Row(nil), rows...) // 拷贝，保持正序

	// 统计
	cor := [3]int{}
	cor2 := [3]int{}
	all6 := 0
	pc100, p6c100 := 0, 0
	for i, r := range rows {
		if r.HOK {
			cor[0]++
		}
		if r.TOK {
			cor[1]++
		}
		if r.OOK {
			cor[2]++
		}
		if r.H2OK {
			cor2[0]++
		}
		if r.T2OK {
			cor2[1]++
		}
		if r.O2OK {
			cor2[2]++
		}
		if r.All6OK {
			all6++
		}
		if i < 100 {
			if r.AllOK {
				pc100++
			}
			if r.All6OK {
				p6c100++
			}
		}
	}
	n100 := n
	if n100 > 100 {
		n100 = 100
	}

	// 下一期预测（对齐 compute_backtest 尾部：用最新一期数据 + 当前状态）
	lb := draws[total-1]
	b, s, g := lb.B, lb.S, lb.G
	pred := Predict{
		H:  engine.ApplyFB(engine.KillH(b, s, g), st.PHK, engine.HFb, b, s, g),
		T:  engine.ApplyFB(engine.KillT(b, s, g), st.PTK, engine.TFb, b, s, g),
		O:  engine.ApplyFB(engine.KillO(b, s, g, st.OFail, total), st.POK, engine.OFb, b, s, g),
		H2: engine.KillH2(b, s, g),
		T2: engine.KillT2(b, s, g),
		O2: engine.KillO2(b, s, g),
	}

	meta := Meta{
		Total: total, LatestIssue: lb.Issue, LatestDate: lb.Date,
		BacktestN: n,
		AccH:      pct(cor[0], n), AccT: pct(cor[1], n), AccO: pct(cor[2], n),
		ErrH: n - cor[0], ErrT: n - cor[1], ErrO: n - cor[2],
		AccAll: pct(cor[0]+cor[1]+cor[2], n*3),
		AccH2:  pct(cor2[0], n), AccT2: pct(cor2[1], n), AccO2: pct(cor2[2], n),
		ErrH2: n - cor2[0], ErrT2: n - cor2[1], ErrO2: n - cor2[2],
		All6: all6, All6Pct: pct(all6, n),
		PeriodCorrect100: pc100, PeriodN100: n100, AccPeriod100: pct(pc100, n100),
		Period6Correct100: p6c100, Period6Pct100: pct(p6c100, n100),
	}

	return &Result{Meta: meta, Rows: rows, Pred: pred}
}

// WindowStats 单个窗口的统计结果
type WindowStats struct {
	AccH, AccT, AccO, Overall float64
	AccH2, AccT2, AccO2       float64
	All6Pct                   float64
	MaxConsecutive            int
}

// MultiWindow 多窗口回测：
// 每个窗口独立重置状态机（窗口第一期 prev 为 None）。
func MultiWindow(draws []data.Draw, windows []int) map[string]WindowStats {
	total := len(draws)
	out := map[string]WindowStats{}
	for _, w := range windows {
		if total <= w {
			continue
		}
		start := total - w
		st := engine.NewState()
		cor := [3]int{}
		cor2 := [3]int{}
		all6 := 0
		maxCons, curCons := 0, 0
		for i := start; i < total; i++ {
			p := draws[i-1]
			d := draws[i]
			hk, tk, ok, hk2, tk2, ok2 := st.Next(p.B, p.S, p.G, d.G)
			hOK, tOK, oOK := hk != d.B, tk != d.S, ok != d.G
			if hOK {
				cor[0]++
			}
			if tOK {
				cor[1]++
			}
			if oOK {
				cor[2]++
			}
			if hk2 != d.B {
				cor2[0]++
			}
			if tk2 != d.S {
				cor2[1]++
			}
			if ok2 != d.G {
				cor2[2]++
			}
			if hOK && tOK && oOK && hk2 != d.B && tk2 != d.S && ok2 != d.G {
				all6++
			}
			if hOK && tOK && oOK {
				curCons = 0
			} else {
				curCons++
				if curCons > maxCons {
					maxCons = curCons
				}
			}
		}
		out[itoa(w)+"期"] = WindowStats{
			AccH: pct(cor[0], w), AccT: pct(cor[1], w), AccO: pct(cor[2], w),
			Overall: pct(cor[0]+cor[1]+cor[2], w*3),
			AccH2:   pct(cor2[0], w), AccT2: pct(cor2[1], w), AccO2: pct(cor2[2], w),
			All6Pct: pct(all6, w), MaxConsecutive: maxCons,
		}
	}
	return out
}

// 随机基线：每位置杀 2 码不中的概率 8/10，三位置 (8/10)^3 ≈ 51.2%。
// 与页面展示口径一致，walk-forward 对照基准。
const BasePct = 0.512

// WFWindow walk-forward 单窗口验证结果
type WFWindow struct {
	Label   string  // 如 "100期"
	N       int     // 窗口期数
	All6    int     // 6 杀全中命中数
	All6Pct float64 // 命中率 %
	BeatPP  float64 // 超越基线幅度 pp
	Z       float64 // 正态近似 z 分数
	PVal    float64 // 双侧 p 值
}

// WalkForward 滚动前推验证（无泄漏）：
// 每个窗口独立重置状态机，仅用窗口内数据逐期预测并对照实际；
// 与随机基线 BasePct 做二项检验（正态近似），给出显著性。
func WalkForward(draws []data.Draw, windows []int) []WFWindow {
	total := len(draws)
	out := []WFWindow{}
	for _, w := range windows {
		if total <= w {
			continue
		}
		start := total - w
		st := engine.NewState()
		all6 := 0
		for i := start; i < total; i++ {
			p := draws[i-1]
			d := draws[i]
			hk, tk, ok, hk2, tk2, ok2 := st.Next(p.B, p.S, p.G, d.G)
			if hk != d.B && tk != d.S && ok != d.G && hk2 != d.B && tk2 != d.S && ok2 != d.G {
				all6++
			}
		}
		pctVal := pct(all6, w)
		se := math.Sqrt(BasePct * (1 - BasePct) / float64(w))
		z := 0.0
		if se > 0 {
			z = (pctVal/100 - BasePct) / se
		}
		out = append(out, WFWindow{
			Label: itoa(w) + "期", N: w, All6: all6, All6Pct: pctVal,
			BeatPP: pctVal - BasePct*100, Z: z, PVal: normP(z),
		})
	}
	return out
}

// normP 双侧正态 p 值（erf 实现）
func normP(z float64) float64 {
	if z < 0 {
		z = -z
	}
	// 双侧: 2*(1-Phi(z)) = erfc(z/sqrt2)
	return math.Erfc(z / math.Sqrt2)
}

func pct(c, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(c) / float64(n) * 100
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func itoa3(a, b, c int) string {
	return itoa(a) + itoa(b) + itoa(c)
}
