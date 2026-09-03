package ssq

import (
	"sort"

	"fc3d-kill6/data"
)

// NumFreq 号码 + 出现次数
type NumFreq struct {
	Num  int
	Freq int
}

// NumMiss 号码 + 遗漏期数
type NumMiss struct {
	Num  int
	Miss int
}

// FreqReds 红球频率榜（按次数降序，同次数按号码升序）
func FreqReds(draws []data.SSQDraw, window int) []NumFreq {
	cnt := freq(draws, window, true)
	return rankFreq(cnt)
}

// FreqBlues 蓝球频率榜（按次数降序）
func FreqBlues(draws []data.SSQDraw, window int) []NumFreq {
	cnt := freq(draws, window, false)
	return rankFreq(cnt)
}

func rankFreq(cnt []int) []NumFreq {
	out := make([]NumFreq, 0, len(cnt)-1)
	for i := 1; i < len(cnt); i++ {
		out = append(out, NumFreq{i, cnt[i]})
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Freq != out[b].Freq {
			return out[a].Freq > out[b].Freq
		}
		return out[a].Num < out[b].Num
	})
	return out
}

// MissReds 红球遗漏榜（按遗漏期数降序，即最久没出的排最前）
func MissReds(draws []data.SSQDraw, window int) []NumMiss {
	m := miss(draws, window, true)
	return rankMiss(m)
}

// MissBlues 蓝球遗漏榜
func MissBlues(draws []data.SSQDraw, window int) []NumMiss {
	m := miss(draws, window, false)
	return rankMiss(m)
}

func rankMiss(m []int) []NumMiss {
	out := make([]NumMiss, 0, len(m)-1)
	for i := 1; i < len(m); i++ {
		out = append(out, NumMiss{i, m[i]})
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Miss != out[b].Miss {
			return out[a].Miss > out[b].Miss
		}
		return out[a].Num < out[b].Num
	})
	return out
}

// SumSeries 最近 window 期红球和值序列（时间正序，最新在末尾）
func SumSeries(draws []data.SSQDraw, window int) []int {
	start := 0
	if window > 0 && window < len(draws) {
		start = len(draws) - window
	}
	out := make([]int, 0, window)
	for _, d := range draws[start:] {
		sum := 0
		for _, r := range d.Reds() {
			sum += r
		}
		out = append(out, sum)
	}
	return out
}
