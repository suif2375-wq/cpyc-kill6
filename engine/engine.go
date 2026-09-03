// Package engine 实现 V9.3 六杀制公式引擎。
// 纯函数 + 有状态回测两种入口，经 golden 基准测试锁定逐期一致性。
package engine

// mod10 转正取模：Go 的 % 对负数返回负数，而 Python 的 % 恒返回非负。
// 原算法中 (b*g-s)%10、(s*g-b)%10 等减法表达式依赖 Python 语义，必须统一走此函数。
func mod10(x int) int {
	r := x % 10
	if r < 0 {
		r += 10
	}
	return r
}

func max3(a, b, c int) int {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

func span(b, s, g int) int { return max3(b, s, g) - min3(b, s, g) }

func mid(b, s, g int) int {
	// 中位数
	if b > s {
		b, s = s, b
	}
	if s > g {
		s, g = g, s
	}
	if b > s {
		b, s = s, b
	}
	return s
}

// ── kill1: 百位 10 条件决策树 ────────────────────────
func KillH(b, s, g int) int {
	if b%2 == 0 && s%2 == 0 && g%2 == 0 {
		return (b + s + g + 1) % 10
	}
	if b%2 == 1 && s%2 == 1 && g%2 == 1 {
		return (b + s + g + 2) % 10
	}
	if b == s {
		return (3 * max3(b, s, g)) % 10
	}
	if b == g {
		return (span(b, s, g) + 1) % 10
	}
	if s == g {
		return (b + s + g + 8) % 10
	}
	if span(b, s, g) == 4 {
		return (b + s + g + 2) % 10
	}
	if span(b, s, g) >= 6 {
		return mod10(b*g - s)
	}
	if (b+s+g)%2 == 1 {
		return (b*b + s + g*g) % 10
	}
	if b < g {
		return (b + s + g + 2) % 10
	}
	if b+s+g <= 12 {
		return (span(b, s, g) + 3) % 10
	}
	return (b + s + g + 1) % 10
}

// ── kill1: 十位 V8a 增强公式法 ───────────────────────
func KillT(b, s, g int) int {
	if (b+s+g)%2 == 1 {
		if (b*b+s*s)%10 == 0 {
			return (b + s + g + 2) % 10 // b²+s²=0 时原公式退化为个位, 致命漏洞修复
		}
		return (b*b + s*s + g) % 10
	}
	if span(b, s, g) >= 6 {
		if b >= s && b >= g { // 百位最大
			return mod10((b + s) * g)
		}
		return (3 * max3(b, s, g)) % 10
	}
	return (g*g + b) % 10
}

// ── kill1: 个位 12 条件决策树 + 自适应备份 ────────────
func oCond(b, s, g int) string {
	sp := span(b, s, g)
	switch {
	case b%2 == 1 && s%2 == 1 && g%2 == 1:
		return "all_odd"
	case b == s:
		return "b_eq_s"
	case b == g:
		return "b_eq_g"
	case s == g:
		return "s_eq_g"
	case sp == 4:
		return "span4"
	case sp == 2:
		return "span2"
	case g == max3(b, s, g):
		return "g_max"
	case b > g:
		return "b_gt_g"
	case b+s+g >= 15:
		return "sum_hi"
	case (b+s+g)%2 == 0:
		return "sum_even"
	case (b+s+g)%2 == 1:
		return "sum_odd"
	default:
		return "default"
	}
}

// killO 计算个位杀码。failState 为 nil 时表示纯函数模式（无自适应）。
func KillO(b, s, g int, failState map[string]int, periodIdx int) int {
	sp := span(b, s, g)
	var pk int
	switch {
	case b%2 == 1 && s%2 == 1 && g%2 == 1:
		pk = (b + s + g + 3) % 10
	case b == s:
		pk = (b + s + g + 6) % 10
	case b == g:
		pk = (b + s + g + 2) % 10
	case s == g:
		pk = (b + s + g + 1) % 10
	case sp == 4:
		pk = (b*b + s*s + g) % 10
	case sp == 2:
		pk = (s*g + b) % 10
	case g == max3(b, s, g):
		pk = (s*g + b) % 10
	case b > g:
		pk = mod10(s * g)
	case b+s+g >= 15:
		pk = (b*s + s*g) % 10
	case (b+s+g)%2 == 0:
		pk = (s*g + b) % 10
	case (b+s+g)%2 == 1:
		pk = (g * g * s) % 10
	default:
		pk = mod10(s*g - b)
	}
	// V8 自适应备份：某条件分支 5 期内失败过 → 切换备份公式
	if failState != nil {
		cn := oCond(b, s, g)
		if last, ok := failState[cn]; ok && periodIdx-last <= OFailWin {
			if fb, ok2 := oBackup[cn]; ok2 {
				pk = mod10(fb(b, s, g))
			}
		}
	}
	return pk
}

// OFailWin 自适应失败窗口（与 Python O_FAIL_WIN=5 一致）
const OFailWin = 5

var oBackup = map[string]func(b, s, g int) int{
	"g_max":   func(b, s, g int) int { return 3 * max3(b, s, g) },
	"b_gt_g":  func(b, s, g int) int { return b*b + g },
	"sum_hi":  func(b, s, g int) int { return b + s + g + 3 },
	"sum_odd": func(b, s, g int) int { return b + s + g + 1 },
	"default": func(b, s, g int) int { return b + s + g + 1 },
}

// ── kill2: V9 独立第二杀码 ───────────────────────────
func KillH2(b, s, g int) int { return mod10(b - span(b, s, g) + 9) }
func KillT2(b, s, g int) int { return mod10(s - mid(b, s, g) + 5) } // s-mid+5 可为负，必须 mod10
func KillO2(b, s, g int) int { return (g*g + abs(b-g)) % 10 }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ── 回退队列（防连续同码）────────────────────────────
// FB 为一个备选公式（输入 b,s,g，返回未取模或已取模值，ApplyFB 统一 mod10）
type FB func(b, s, g int) int

var HFb = []FB{
	func(b, s, g int) int { return (b + s + g + 1) % 10 },
	func(b, s, g int) int { return mod10(b * s) },
}
var TFb = []FB{
	func(b, s, g int) int { return (g*g + b) % 10 },
	func(b, s, g int) int { return (b + s + g + 1) % 10 },
	func(b, s, g int) int { return span(b, s, g) },
	func(b, s, g int) int { return mod10(b * g) },
	func(b, s, g int) int { return (b + s) % 10 },
	func(b, s, g int) int { return mod10(b * s) },
}
var OFb = []FB{
	func(b, s, g int) int { return (b + s + g + 1) % 10 },
	func(b, s, g int) int { return mod10(b * s) },
}

// ApplyFB 与 Python apply_fb 语义一致：
// 当前杀码 != 昨日同位置杀码 → 原样返回；否则依次尝试备选，全失败则 (kill+1)%10。
func ApplyFB(kill, prev int, fbList []FB, b, s, g int) int {
	if kill != prev {
		return kill
	}
	for _, f := range fbList {
		alt := mod10(f(b, s, g))
		if alt != prev {
			return alt
		}
	}
	return (kill + 1) % 10
}

// ── 有状态回测引擎（含回退队列与个位自适应状态机）──
// State 保存跨期状态：昨日三位置杀码 + 个位失败追踪
type State struct {
	PHK, PTK, POK int            // 上一期（第 i-1 期）的三位置杀码；-1 表示无
	HasPrev       bool           // 是否有上一期
	OFail         map[string]int // 个位条件分支最近失败期号
	PeriodIdx     int            // 当前循环期号 i
}

// Next 计算第 i 期（以 draws[i-1] 为上期输入）的 6 杀码，并推进状态。
// b,s,g 为上期开奖号；onesActual 为当期个位实际值（用于失败追踪）。
func (st *State) Next(b, s, g, onesActual int) (hk, tk, ok, hk2, tk2, ok2 int) {
	st.PeriodIdx++
	i := st.PeriodIdx

	// kill1
	hkRaw := KillH(b, s, g)
	if st.HasPrev {
		hk = ApplyFB(hkRaw, st.PHK, HFb, b, s, g)
	} else {
		hk = hkRaw
	}

	tkRaw := KillT(b, s, g)
	if st.HasPrev {
		tk = ApplyFB(tkRaw, st.PTK, TFb, b, s, g)
	} else {
		tk = tkRaw
	}

	okRaw := KillO(b, s, g, st.OFail, i)
	if st.HasPrev {
		ok = ApplyFB(okRaw, st.POK, OFb, b, s, g)
	} else {
		ok = okRaw
	}

	// 个位失败追踪（与 Python: if pok == data[i]["g"]: o_fail[cond]=i）
	if ok == onesActual {
		st.OFail[oCond(b, s, g)] = i
	}

	// kill2
	hk2 = KillH2(b, s, g)
	tk2 = KillT2(b, s, g)
	ok2 = KillO2(b, s, g)

	st.PHK, st.PTK, st.POK = hk, tk, ok
	st.HasPrev = true
	return
}

// NewState 创建初始回测状态
func NewState() *State {
	return &State{PHK: -1, PTK: -1, POK: -1, OFail: map[string]int{}}
}
