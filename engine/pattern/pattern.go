// Package pattern 实现跨期规律路径挖掘引擎。
//
// 算法移植自 https://github.com/liangguifeng/lottery-analyzer（MIT）：
//   - 把历史按「规律期数 + 1」切块，每块 = N 期规律区间 + 1 期验证期；
//   - 跨期取位置组合（如 "1_1|2_2|3_1" 表示第 1 期第 1 位 + 第 2 期第 2 位 + 第 3 期第 1 位）；
//   - 前 MinConsecutive 个块连续命中即为稳定规律，应用到最新窗口预测下一期。
//
// 支持 5 种分析器：胆码（至少一位开出）/ 毒胆（全部不开）/ 和尾杀号 ×3。
// 严格无未来信息：预测第 i 期只使用 draws[0:i]，回测按期号滚动回溯。
package pattern

import (
	"sort"
	"strconv"
	"strings"
)

// Kind 分析器类型（对齐 lottery-analyzer 的 5 种分析器）
type Kind int

const (
	Danma Kind = iota // 胆码：规律路径数字至少一位开出
	Dudan             // 毒胆：规律路径数字全部不开
	SumBH             // 杀百个和尾：组合和尾 ≠ (百+个)和尾
	SumBT             // 杀百十和尾：组合和尾 ≠ (百+十)和尾
	SumTO             // 杀十个和尾：组合和尾 ≠ (十+个)和尾
)

// String 完整名称
func (k Kind) String() string {
	switch k {
	case Danma:
		return "胆码分析"
	case Dudan:
		return "毒胆分析"
	case SumBH:
		return "杀百个和尾"
	case SumBT:
		return "杀百十和尾"
	case SumTO:
		return "杀十个和尾"
	}
	return "未知分析器"
}

// Short 卡片短名
func (k Kind) Short() string {
	switch k {
	case Danma:
		return "胆码"
	case Dudan:
		return "毒胆"
	case SumBH:
		return "百个和尾"
	case SumBT:
		return "百十和尾"
	case SumTO:
		return "十个和尾"
	}
	return "?"
}

// Kinds 全部分析器（固定顺序）
var Kinds = []Kind{Danma, Dudan, SumBH, SumBT, SumTO}

// Draw 一期开奖（时间正序）
type Draw struct {
	Issue   string
	B, S, G int
}

// Config 分析参数（对齐 PHP 默认值 analyzePeriods=3 minConsecutive=3 combinationSize=3 intervalPeriods=0）
type Config struct {
	Periods        int // 规律期数
	MinConsecutive int // 最小连续命中期数
	CombSize       int // 组合大小
	Interval       int // 间隔期数（保留字段，默认 0）
}

// Default 默认参数
var Default = Config{Periods: 3, MinConsecutive: 3, CombSize: 3, Interval: 0}

func (c Config) norm() Config {
	if c.Periods <= 0 {
		c.Periods = Default.Periods
	}
	if c.MinConsecutive <= 0 {
		c.MinConsecutive = Default.MinConsecutive
	}
	if c.CombSize <= 0 {
		c.CombSize = Default.CombSize
	}
	return c
}

// Hit 一条命中规律
type Hit struct {
	Path    string // 规律路径 "1_1|2_2|3_1"
	Values  []int  // 组合数字（验证窗口内取值）
	MaxCons int    // 最大连续命中块数（从最新块起算）
	Next    []int  // 预测数字（应用到最新窗口；和尾分析时为杀尾数字）
}

// NumFreq 数字频次
type NumFreq struct {
	Num  int
	Freq int
}

// Analysis 一次完整分析结果
type Analysis struct {
	Kind     Kind
	Config   Config
	HitCount int       // 命中规律数
	Hits     []Hit     // 规律列表（MaxCons 降序）
	TopNums  []NumFreq // 预测数字频次（全规律聚合，降序）
	Picks    []int     // 本期推荐：胆码/毒胆 = 高频数字 Top3；和尾 = 频次最高杀尾
}

// ── 组合生成 ──────────────────────────────────────────

type comboItem struct {
	group int // 1..Periods
	pos   int // 1..3 (1=百 2=十 3=个)
}

func comboKey(items []comboItem) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = strconv.Itoa(it.group) + "_" + strconv.Itoa(it.pos)
	}
	return strings.Join(parts, "|")
}

// buildCombos 预生成全部跨期组合（与 PHP generateCrossGroupCombinations 同序：
// group 升序、组内 pos 升序，key 规范唯一）。
func buildCombos(periods, combSize int) [][]comboItem {
	items := make([]comboItem, 0, periods*3)
	for g := 1; g <= periods; g++ {
		for p := 1; p <= 3; p++ {
			items = append(items, comboItem{group: g, pos: p})
		}
	}
	total := len(items)
	if combSize <= 0 || combSize > total {
		return nil
	}
	idx := make([]int, combSize)
	for i := range idx {
		idx[i] = i
	}
	out := make([][]comboItem, 0, 256)
	for {
		comb := make([]comboItem, combSize)
		for i, ix := range idx {
			comb[i] = items[ix]
		}
		out = append(out, comb)
		i := combSize - 1
		for i >= 0 && idx[i] == total-(combSize-i) {
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

// ── 命中判定 ──────────────────────────────────────────

// comboValues 取组合在窗口中的数字：窗口 win 时间正序，group g 期 = win[g-1]。
func comboValues(items []comboItem, win [][3]int) []int {
	vals := make([]int, len(items))
	for i, it := range items {
		d := win[it.group-1]
		switch it.pos {
		case 1:
			vals[i] = d[0]
		case 2:
			vals[i] = d[1]
		default:
			vals[i] = d[2]
		}
	}
	return vals
}

// match 判定组合是否命中验证期（按分析器类型）
func match(kind Kind, values []int, verify [3]int) bool {
	switch kind {
	case Danma:
		return hasIntersect(values, verify)
	case Dudan:
		return !hasIntersect(values, verify)
	case SumBH:
		return sumTail(values) != (verify[0]+verify[2])%10
	case SumBT:
		return sumTail(values) != (verify[0]+verify[1])%10
	case SumTO:
		return sumTail(values) != (verify[1]+verify[2])%10
	}
	return false
}

func sumTail(vals []int) int {
	t := 0
	for _, v := range vals {
		t += v
	}
	return t % 10
}

func hasIntersect(a []int, b [3]int) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

// ── 引擎 ──────────────────────────────────────────────

// Engine 复用预生成组合的规律挖掘引擎
type Engine struct {
	cfg    Config
	combos [][]comboItem
}

// New 创建引擎
func New(cfg Config) *Engine {
	cfg = cfg.norm()
	return &Engine{cfg: cfg, combos: buildCombos(cfg.Periods, cfg.CombSize)}
}

// Analyze 用 draws（时间正序，不含未来信息）挖掘规律并给出下一期推荐。
func Analyze(kind Kind, draws []Draw, cfg Config) *Analysis {
	return New(cfg).Analyze(kind, draws)
}

// Analyze 用 draws 挖掘规律并给出下一期推荐。
func (e *Engine) Analyze(kind Kind, draws []Draw) *Analysis {
	cfg := e.cfg
	res := &Analysis{Kind: kind, Config: cfg}
	n := len(draws)
	block := cfg.Periods + cfg.Interval + 1
	need := cfg.Periods + block*cfg.MinConsecutive
	if n < need || len(e.combos) == 0 {
		return res
	}
	adLen := n - cfg.Periods // 剔除最新 Periods 期（作为预测窗口）
	nBlocks := adLen / block
	if nBlocks < cfg.MinConsecutive {
		return res
	}

	// 候选：前 MinConsecutive 个块（从最新往旧）全部命中
	var cands [][]comboItem
	for _, comb := range e.combos {
		ok := true
		for j := 0; j < cfg.MinConsecutive; j++ {
			vals := comboValuesAt(comb, draws, adLen-(j+1)*block, cfg.Periods)
			if !match(kind, vals, draws[adLen-j*block-1].arr()) {
				ok = false
				break
			}
		}
		if ok {
			cands = append(cands, comb)
		}
	}

	// 预测窗口 = 最新 Periods 期（已开奖，无未来信息）
	freq := map[int]int{} // 预测数字频次
	tailFreq := map[int]int{}
	for _, comb := range cands {
		vals := comboValuesAt(comb, draws, n-cfg.Periods, cfg.Periods)
		var next []int
		if kind == SumBH || kind == SumBT || kind == SumTO {
			t := sumTail(vals)
			next = []int{t}
			tailFreq[t]++
		} else {
			next = make([]int, len(vals))
			copy(next, vals)
			for _, v := range vals {
				freq[v]++
			}
		}
		h := Hit{Path: comboKey(comb), Values: vals, Next: next}
		h.MaxCons = e.maxConsecutive(kind, comb, draws, adLen, block)
		res.Hits = append(res.Hits, h)
	}

	sort.Slice(res.Hits, func(i, j int) bool {
		if res.Hits[i].MaxCons != res.Hits[j].MaxCons {
			return res.Hits[i].MaxCons > res.Hits[j].MaxCons
		}
		return res.Hits[i].Path < res.Hits[j].Path
	})

	res.HitCount = len(res.Hits)
	if kind == SumBH || kind == SumBT || kind == SumTO {
		res.TopNums = freqToTop(tailFreq, 0)
		res.Picks = freqToNums(tailFreq, 1)
	} else {
		res.TopNums = freqToTop(freq, 0)
		res.Picks = freqToNums(freq, 3)
	}
	return res
}

// maxConsecutive 从最新块起连续命中的块数
func (e *Engine) maxConsecutive(kind Kind, comb []comboItem, draws []Draw, adLen, block int) int {
	cnt := 0
	for j := 0; (j+1)*block <= adLen; j++ {
		vals := comboValuesAt(comb, draws, adLen-(j+1)*block, e.cfg.Periods)
		if !match(kind, vals, draws[adLen-j*block-1].arr()) {
			break
		}
		cnt++
	}
	return cnt
}

// comboValuesAt 取组合在 draws[base:base+Periods] 窗口中的数字
func comboValuesAt(items []comboItem, draws []Draw, base, periods int) []int {
	vals := make([]int, len(items))
	for i, it := range items {
		d := draws[base+it.group-1]
		switch it.pos {
		case 1:
			vals[i] = d.B
		case 2:
			vals[i] = d.S
		default:
			vals[i] = d.G
		}
	}
	return vals
}

func (d Draw) arr() [3]int { return [3]int{d.B, d.S, d.G} }

// ── 频次聚合 ──────────────────────────────────────────

func freqToNums(freq map[int]int, n int) []int {
	top := freqToTop(freq, n)
	out := make([]int, len(top))
	for i, t := range top {
		out[i] = t.Num
	}
	return out
}

func freqToTop(freq map[int]int, n int) []NumFreq {
	out := make([]NumFreq, 0, len(freq))
	for num, f := range freq {
		out = append(out, NumFreq{Num: num, Freq: f})
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

// ── 滚动回测 ──────────────────────────────────────────

// KindStats 单个分析器的回测统计
type KindStats struct {
	Kind     Kind
	N        int     // 窗口期数（近 Window 期）
	Hit      int     // 窗口命中
	Rate     float64 // 窗口命中率 %
	Base     float64 // 窗口基线 %
	FullN    int     // 全量回测期数
	FullHit  int
	FullRate float64
	FullBase float64
}

// BacktestResult 全部 5 种分析器的滚动回测 + 最新一期分析
type BacktestResult struct {
	Window int
	Stats  []KindStats
	Latest map[Kind]*Analysis // 最新一期分析（页面展示）
}

// Stat 返回指定分析器的回测统计
func (r *BacktestResult) Stat(k Kind) *KindStats {
	for i := range r.Stats {
		if r.Stats[i].Kind == k {
			return &r.Stats[i]
		}
	}
	return nil
}

// Backtest 滚动回测（窗口独立重置，无未来信息）：用 draws[0:i] 预测第 i 期。
func Backtest(draws []Draw, cfg Config, window int) *BacktestResult {
	return New(cfg).Backtest(draws, window)
}

// Backtest 滚动回测
func (e *Engine) Backtest(draws []Draw, window int) *BacktestResult {
	cfg := e.cfg
	block := cfg.Periods + cfg.Interval + 1
	need := cfg.Periods + block*cfg.MinConsecutive
	n := len(draws)
	if window <= 0 {
		window = 100
	}
	res := &BacktestResult{Window: window, Latest: map[Kind]*Analysis{}}
	for _, k := range Kinds {
		res.Stats = append(res.Stats, KindStats{Kind: k})
	}
	if n < need {
		return res
	}

	// 逐期滚动：预测第 i 期（用 draws[0:i]），对照实际开奖
	type acc struct {
		hit, fullHit   int
		base, fullBase float64
	}
	accs := make(map[Kind]*acc, len(Kinds))
	for _, k := range Kinds {
		accs[k] = &acc{}
	}
	wStart := n - window
	if wStart < need {
		wStart = need
	}
	for i := need; i < n; i++ {
		act := draws[i].arr()
		for _, k := range Kinds {
			picks := e.picks(k, draws[:i])
			hit := picksHit(k, picks, act)
			base := picksBase(k, picks)
			ac := accs[k]
			ac.fullHit += b2i(hit)
			ac.fullBase += base
			if i >= wStart {
				ac.hit += b2i(hit)
				ac.base += base
			}
		}
	}
	for idx, k := range Kinds {
		ac := accs[k]
		wn := n - wStart
		res.Stats[idx].N = wn
		res.Stats[idx].Hit = ac.hit
		res.Stats[idx].Rate = pct(ac.hit, wn)
		res.Stats[idx].Base = pctAvg(ac.base, wn)
		res.Stats[idx].FullN = n - need
		res.Stats[idx].FullHit = ac.fullHit
		res.Stats[idx].FullRate = pct(ac.fullHit, n-need)
		res.Stats[idx].FullBase = pctAvg(ac.fullBase, n-need)
	}
	for _, k := range Kinds {
		res.Latest[k] = e.Analyze(k, draws)
	}
	return res
}

// picks 快速获取本期推荐（无 Hits 明细，供回测循环使用）
func (e *Engine) picks(kind Kind, draws []Draw) []int {
	return e.Analyze(kind, draws).Picks
}

// picksHit 判断本期推荐是否命中实际开奖
func picksHit(k Kind, picks []int, act [3]int) bool {
	switch k {
	case Danma:
		return hasIntersect(picks, act)
	case Dudan:
		return !hasIntersect(picks, act)
	default: // SumTail 系列：杀尾 ≠ 实际和尾
		if len(picks) == 0 {
			return false
		}
		return picks[0] != (act[0]+act[1]+act[2])%10
	}
}

// picksBase 本期推荐的随机基线（按推荐数字去重数量）
func picksBase(k Kind, picks []int) float64 {
	kk := float64(len(picks))
	switch k {
	case Danma: // k 个不同数字，3 个位置至少中 1
		return 1 - pow3((10-kk)/10)
	case Dudan: // k 个不同数字，3 个位置全避开
		return pow3((10 - kk) / 10)
	default: // 和尾 0-9 均匀，避开 k 个尾
		return 1 - kk/10
	}
}

func pow3(x float64) float64 { return x * x * x }

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func pct(c, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(c) / float64(n) * 100
}

func pctAvg(sum float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n) * 100
}
