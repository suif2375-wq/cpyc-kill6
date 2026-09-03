package pattern

import (
	"sort"
	"strconv"
	"strings"

	"fc3d-kill6/data"
)

// DigitKind 是 lottery-analyzer 规律引擎在任意位数彩票上的通用分析器。
type DigitKind int

const (
	DigitDanma   DigitKind = iota // 组合数字至少有一位在验证期出现
	DigitDudan                    // 组合数字在验证期全部不出现
	DigitSumTail                  // 组合和值尾不等于验证期总和尾
)

// DigitConfig 对齐 lottery-analyzer 的四个核心参数。
type DigitConfig struct {
	Periods        int
	MinConsecutive int
	CombSize       int
	Interval       int
}

var DefaultDigitConfig = DigitConfig{Periods: 3, MinConsecutive: 3, CombSize: 3}

func (c DigitConfig) norm() DigitConfig {
	if c.Periods <= 0 {
		c.Periods = DefaultDigitConfig.Periods
	}
	if c.MinConsecutive <= 0 {
		c.MinConsecutive = DefaultDigitConfig.MinConsecutive
	}
	if c.CombSize <= 0 {
		c.CombSize = DefaultDigitConfig.CombSize
	}
	if c.Interval < 0 {
		c.Interval = 0
	}
	return c
}

// DigitHit 是一条跨期规律路径。
type DigitHit struct {
	Path    string
	Values  []int
	MaxCons int
	Next    []int
}

// DigitNumFreq 是规律聚合后的数字频次。
type DigitNumFreq struct {
	Num  int
	Freq int
}

// DigitAnalysis 是一次通用规律分析结果。
type DigitAnalysis struct {
	Kind     DigitKind
	Config   DigitConfig
	HitCount int
	Hits     []DigitHit
	TopNums  []DigitNumFreq
	Picks    []int
}

type digitComboItem struct {
	group int
	pos   int
}

// AnalyzeDigits 移植 lottery-analyzer 的跨期规律算法，支持排列3/排列5。
// draws 必须按期号从旧到新排列；函数只读取传入的历史数据。
func AnalyzeDigits(kind DigitKind, draws []data.DigitDraw, cfg DigitConfig) *DigitAnalysis {
	cfg = cfg.norm()
	res := &DigitAnalysis{Kind: kind, Config: cfg}
	positions := digitPositions(draws)
	if positions == 0 || len(draws) == 0 {
		return res
	}
	combos := buildDigitCombos(cfg.Periods, positions, cfg.CombSize)
	block := cfg.Periods + cfg.Interval + 1
	need := cfg.Periods + block*cfg.MinConsecutive
	if len(draws) < need || len(combos) == 0 {
		return res
	}
	adLen := len(draws) - cfg.Periods
	var candidates [][]digitComboItem
	for _, combo := range combos {
		ok := true
		for j := 0; j < cfg.MinConsecutive; j++ {
			vals := digitComboValuesAt(combo, draws, adLen-(j+1)*block)
			verify := draws[adLen-j*block-1].Digits
			if !digitMatch(kind, vals, verify) {
				ok = false
				break
			}
		}
		if ok {
			candidates = append(candidates, combo)
		}
	}

	freq := map[int]int{}
	tailFreq := map[int]int{}
	for _, combo := range candidates {
		vals := digitComboValuesAt(combo, draws, len(draws)-cfg.Periods)
		next := append([]int(nil), vals...)
		if kind == DigitSumTail {
			next = []int{digitSumTail(vals)}
			tailFreq[next[0]]++
		} else {
			for _, v := range vals {
				freq[v]++
			}
		}
		hit := DigitHit{Path: digitComboKey(combo), Values: vals, Next: next}
		hit.MaxCons = digitMaxConsecutive(kind, combo, draws, adLen, block, cfg.Periods)
		res.Hits = append(res.Hits, hit)
	}
	sort.Slice(res.Hits, func(i, j int) bool {
		if res.Hits[i].MaxCons != res.Hits[j].MaxCons {
			return res.Hits[i].MaxCons > res.Hits[j].MaxCons
		}
		return res.Hits[i].Path < res.Hits[j].Path
	})
	res.HitCount = len(res.Hits)
	if kind == DigitSumTail {
		res.TopNums = digitFreqToTop(tailFreq, 0)
		res.Picks = digitFreqToNums(tailFreq, 1)
	} else {
		res.TopNums = digitFreqToTop(freq, 0)
		res.Picks = digitFreqToNums(freq, 3)
	}
	return res
}

func digitPositions(draws []data.DigitDraw) int {
	if len(draws) == 0 {
		return 0
	}
	return len(draws[0].Digits)
}

func buildDigitCombos(periods, positions, combSize int) [][]digitComboItem {
	if periods <= 0 || positions <= 0 || combSize <= 0 || combSize > periods*positions {
		return nil
	}
	items := make([]digitComboItem, 0, periods*positions)
	for g := 1; g <= periods; g++ {
		for p := 1; p <= positions; p++ {
			items = append(items, digitComboItem{group: g, pos: p})
		}
	}
	idx := make([]int, combSize)
	for i := range idx {
		idx[i] = i
	}
	out := make([][]digitComboItem, 0)
	for {
		combo := make([]digitComboItem, combSize)
		for i, ix := range idx {
			combo[i] = items[ix]
		}
		out = append(out, combo)
		i := combSize - 1
		for i >= 0 && idx[i] == len(items)-(combSize-i) {
			i--
		}
		if i < 0 {
			break
		}
		idx[i]++
		for j := i + 1; j < combSize; j++ {
			idx[j] = idx[j-1] + 1
		}
	}
	return out
}

func digitComboKey(combo []digitComboItem) string {
	parts := make([]string, len(combo))
	for i, item := range combo {
		parts[i] = strconv.Itoa(item.group) + "_" + strconv.Itoa(item.pos)
	}
	return strings.Join(parts, "|")
}

func digitComboValuesAt(combo []digitComboItem, draws []data.DigitDraw, base int) []int {
	vals := make([]int, len(combo))
	for i, item := range combo {
		idx := base + item.group - 1
		if idx < 0 || idx >= len(draws) || item.pos <= 0 || item.pos > len(draws[idx].Digits) {
			continue
		}
		vals[i] = draws[idx].Digits[item.pos-1]
	}
	return vals
}

func digitMatch(kind DigitKind, values, verify []int) bool {
	switch kind {
	case DigitDanma:
		return digitIntersects(values, verify)
	case DigitDudan:
		return !digitIntersects(values, verify)
	case DigitSumTail:
		return digitSumTail(values) != digitSumTail(verify)
	default:
		return false
	}
}

func digitIntersects(a, b []int) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

func digitSumTail(values []int) int {
	t := 0
	for _, v := range values {
		t += v
	}
	return t % 10
}

func digitMaxConsecutive(kind DigitKind, combo []digitComboItem, draws []data.DigitDraw, adLen, block, periods int) int {
	cnt := 0
	for j := 0; (j+1)*block <= adLen; j++ {
		vals := digitComboValuesAt(combo, draws, adLen-(j+1)*block)
		if !digitMatch(kind, vals, draws[adLen-j*block-1].Digits) {
			break
		}
		cnt++
	}
	return cnt
}

func digitFreqToNums(freq map[int]int, n int) []int {
	top := digitFreqToTop(freq, n)
	out := make([]int, len(top))
	for i, item := range top {
		out[i] = item.Num
	}
	return out
}

func digitFreqToTop(freq map[int]int, n int) []DigitNumFreq {
	out := make([]DigitNumFreq, 0, len(freq))
	for num, f := range freq {
		out = append(out, DigitNumFreq{Num: num, Freq: f})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Freq != out[j].Freq {
			return out[i].Freq > out[j].Freq
		}
		return out[i].Num < out[j].Num
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}
