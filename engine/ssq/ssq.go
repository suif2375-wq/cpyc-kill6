// Package ssq 实现双色球杀号引擎（纯函数，可回测）。
//
// 双色球号码空间：红球 1-33（每期开 6 个不重复），蓝球 1-16（每期开 1 个）。
// "杀号"= 基于历史窗口统计，排除 N 个最不可能开出的红球 + M 个最不可能开出的蓝球。
// 三种候选策略（冷/热/遗漏），具体采用哪种由回测数据决定。
package ssq

import (
	"sort"

	"fc3d-kill6/data"
)

// Strategy 杀号策略
type Strategy int

const (
	StrategyCold Strategy = iota // 最冷：近 window 期出现次数最少的号码
	StrategyHot                  // 最热：近 window 期出现次数最多的号码
	StrategyMiss                 // 遗漏：距离上次开出最久（期数最长）
)

// StrName 策略中文名
func (s Strategy) StrName() string {
	switch s {
	case StrategyCold:
		return "冷号法"
	case StrategyHot:
		return "热号法"
	case StrategyMiss:
		return "遗漏法"
	}
	return "未知"
}

// freq 统计窗口内号码出现次数（red=true 为红球 1-33，false 为蓝球 1-16）
func freq(draws []data.SSQDraw, window int, red bool) []int {
	maxN := 16
	if red {
		maxN = 33
	}
	cnt := make([]int, maxN+1)
	start := 0
	if window > 0 && window < len(draws) {
		start = len(draws) - window
	}
	for _, d := range draws[start:] {
		if red {
			for _, r := range d.Reds() {
				cnt[r]++
			}
		} else {
			cnt[d.Blue]++
		}
	}
	return cnt
}

// miss 统计窗口内各号码距窗口末尾的遗漏期数（red=true 红球，false 蓝球）
func miss(draws []data.SSQDraw, window int, red bool) []int {
	maxN := 16
	if red {
		maxN = 33
	}
	last := make([]int, maxN+1)
	start := 0
	if window > 0 && window < len(draws) {
		start = len(draws) - window
	}
	for i := start; i < len(draws); i++ {
		d := draws[i]
		if red {
			for _, r := range d.Reds() {
				last[r] = i
			}
		} else if d.Blue >= 1 && d.Blue <= 16 {
			last[d.Blue] = i
		}
	}
	end := len(draws) - 1
	out := make([]int, maxN+1)
	for n := 1; n <= maxN; n++ {
		if last[n] == 0 && !appeared(draws, start, n, red) {
			out[n] = end - start + 1 // 窗口内从未出现，视为整窗遗漏
		} else {
			out[n] = end - last[n]
		}
	}
	return out
}

func appeared(draws []data.SSQDraw, start, n int, red bool) bool {
	for i := start; i < len(draws); i++ {
		if red {
			if draws[i].HasRed(n) {
				return true
			}
		} else if draws[i].Blue == n {
			return true
		}
	}
	return false
}

// pickLowest 选 score 最小的 n 个号码（并列按号码小优先）
func pickLowest(score []int, n int) []int {
	type pair struct{ num, v int }
	ps := make([]pair, 0, len(score)-1)
	for i := 1; i < len(score); i++ {
		ps = append(ps, pair{i, score[i]})
	}
	sort.SliceStable(ps, func(a, b int) bool {
		if ps[a].v != ps[b].v {
			return ps[a].v < ps[b].v
		}
		return ps[a].num < ps[b].num
	})
	out := make([]int, 0, n)
	for i := 0; i < n && i < len(ps); i++ {
		out = append(out, ps[i].num)
	}
	return out
}

// pickHighest 选 score 最大的 n 个号码（并列按号码大优先）
func pickHighest(score []int, n int) []int {
	type pair struct{ num, v int }
	ps := make([]pair, 0, len(score)-1)
	for i := 1; i < len(score); i++ {
		ps = append(ps, pair{i, score[i]})
	}
	sort.SliceStable(ps, func(a, b int) bool {
		if ps[a].v != ps[b].v {
			return ps[a].v > ps[b].v
		}
		return ps[a].num > ps[b].num
	})
	out := make([]int, 0, n)
	for i := 0; i < n && i < len(ps); i++ {
		out = append(out, ps[i].num)
	}
	return out
}

// KillReds 基于最近 window 期数据，按策略选出 n 个最不可能开出的红球。
// draws 按时间正序（最新在末尾）。window<=0 或超界时用全部数据。
func KillReds(draws []data.SSQDraw, n, window int, s Strategy) []int {
	switch s {
	case StrategyHot:
		return pickHighest(freq(draws, window, true), n)
	case StrategyMiss:
		return pickLowest(miss(draws, window, true), n)
	default:
		return pickLowest(freq(draws, window, true), n)
	}
}

// KillBlues 基于最近 window 期数据，按策略选出 n 个最不可能开出的蓝球。
func KillBlues(draws []data.SSQDraw, n, window int, s Strategy) []int {
	switch s {
	case StrategyHot:
		return pickHighest(freq(draws, window, false), n)
	case StrategyMiss:
		return pickLowest(miss(draws, window, false), n)
	default:
		return pickLowest(freq(draws, window, false), n)
	}
}
