// lotterymcp_probe 对 Lotterymcp 仓库中公开的排列3/排列5分析器做本地滚动回测。
// 代码按 examples/python/common/three_digit_analysis.py 与
// positional_sequence_analysis.py 的评分公式翻译，输入使用本项目 CSV。
package main

import (
	"flag"
	"fmt"
	"math"
	"sort"

	"fc3d-kill6/data"
)

type scored struct {
	d []int
	s float64
}

func main() {
	p3Path := flag.String("p3-csv", "p3-history.csv", "排列3 CSV")
	p5Path := flag.String("p5-csv", "p5-history.csv", "排列5 CSV")
	window := flag.Int("window", 1000, "最近回测期数")
	top := flag.Int("top", 10, "每期推荐组数")
	flag.Parse()
	p3, err := data.LoadDigitCSV(*p3Path, 3)
	if err != nil {
		panic(err)
	}
	p5, err := data.LoadDigitCSV(*p5Path, 5)
	if err != nil {
		panic(err)
	}
	probe("Lotterymcp three_digit_analysis · 排列3", p3, *window, *top, 3, p3Candidates)
	probe("Lotterymcp positional_sequence_analysis · 排列5", p5, *window, *top, 5, p5Candidates)
}

func probe(name string, draws []data.DigitDraw, window, top, positions int, fn func([]data.DigitDraw, int) []scored) {
	start := 1
	if len(draws) > window {
		start = len(draws) - window
	}
	warmup := 80
	if start < warmup {
		start = warmup
	}
	exact, two, n := 0, 0, 0
	posHit := make([]int, positions)
	for t := start; t < len(draws); t++ {
		pred := fn(draws[:t], top)
		if len(pred) == 0 {
			continue
		}
		actual := draws[t].Digits
		oneExact, oneTwo := false, false
		for _, item := range pred {
			matches := 0
			for p := 0; p < positions; p++ {
				if item.d[p] == actual[p] {
					matches++
					posHit[p]++
				}
			}
			if matches == positions {
				oneExact = true
			}
			if matches >= 2 {
				oneTwo = true
			}
		}
		if oneExact {
			exact++
		}
		if oneTwo {
			two++
		}
		n++
	}
	fmt.Printf("%s: n=%d exact@%d=%.2f%% two+@%d=%.2f%%", name, n, top, pct(exact, n), top, pct(two, n))
	for p, h := range posHit {
		fmt.Printf(" pos%d=%.2f%%", p+1, pct(h, n*top))
	}
	if len(draws) > 0 {
		latest := fn(draws, top)
		fmt.Printf(" latest=")
		for i, item := range latest {
			if i > 0 {
				fmt.Print(",")
			}
			for _, d := range item.d {
				fmt.Print(d)
			}
		}
	}
	fmt.Println()
}

func p3Candidates(draws []data.DigitDraw, top int) []scored {
	if len(draws) == 0 {
		return nil
	}
	newest := reverse(draws)
	mat := make([][10][10]float64, 3)
	for p := 0; p < 3; p++ {
		for i := 0; i < 10; i++ {
			for j := 0; j < 10; j++ {
				mat[p][i][j] = 1
			}
		}
	}
	var pos [3][10]float64
	var sumF, spanF, oddF, typeF [100]float64
	for i, row := range newest {
		if len(row.Digits) < 3 {
			continue
		}
		if i > 0 {
			prev := newest[i-1].Digits
			for p := 0; p < 3; p++ {
				mat[p][prev[p]][row.Digits[p]]++
			}
		}
		d := row.Digits[:3]
		for p := 0; p < 3; p++ {
			pos[p][d[p]]++
		}
		sumF[sum3(d)]++
		spanF[max3(d)-min3(d)]++
		oddF[oddEvenKey(d)]++
		typeF[typeKey(d)]++
	}
	last := newest[0].Digits
	out := make([]scored, 0, 1000)
	for a := 0; a < 10; a++ {
		for b := 0; b < 10; b++ {
			for c := 0; c < 10; c++ {
				d := []int{a, b, c}
				markov, positional := 0.0, 0.0
				for p, v := range d {
					rowTotal := 0.0
					for j := 0; j < 10; j++ {
						rowTotal += mat[p][last[p]][j]
					}
					markov += mat[p][last[p]][v] / rowTotal
					positional += pos[p][v] / float64(len(newest))
				}
				markov /= 3
				positional /= 3
				total := sum3(d)
				s := markov*0.35 + positional*0.25 + sumF[total]/float64(len(newest))*0.15 + spanF[max3(d)-min3(d)]/float64(len(newest))*0.10 + oddF[oddEvenKey(d)]/float64(len(newest))*0.10 + typeF[typeKey(d)]/float64(len(newest))*0.05
				out = append(out, scored{d: d, s: s})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].s > out[j].s })
	if top > len(out) {
		top = len(out)
	}
	return out[:top]
}

func p5Candidates(draws []data.DigitDraw, top int) []scored {
	if len(draws) == 0 {
		return nil
	}
	newest := reverse(draws)
	positions := 5
	var global, recent [5][10]float64
	var mat [5][10][10]float64
	for p := 0; p < positions; p++ {
		for i := 0; i < 10; i++ {
			for j := 0; j < 10; j++ {
				mat[p][i][j] = 1
			}
		}
	}
	var sumF, uniqueF, oddF [100]float64
	for i, row := range newest {
		if len(row.Digits) < positions {
			continue
		}
		if i > 0 {
			prev := newest[i-1].Digits
			for p := 0; p < positions; p++ {
				mat[p][prev[p]][row.Digits[p]]++
			}
		}
		for p := 0; p < positions; p++ {
			global[p][row.Digits[p]]++
			if i < 40 {
				recent[p][row.Digits[p]]++
			}
		}
		sumF[sumDigits(row.Digits)]++
		uniqueF[len(uniqueDigits(row.Digits))]++
		oddF[oddEvenKey(row.Digits)]++
	}
	pool := make([][]int, positions)
	for p := 0; p < positions; p++ {
		pool[p] = topDigits(global[p], 3)
		for _, d := range topDigits(recent[p], 3) {
			if !containsInt(pool[p], d) {
				pool[p] = append(pool[p], d)
			}
		}
		if len(pool[p]) > 3 {
			pool[p] = pool[p][:3]
		}
	}
	last := newest[0].Digits
	out := make([]scored, 0, 243)
	cur := make([]int, positions)
	var walk func(int)
	walk = func(p int) {
		if p == positions {
			markov, positional := 0.0, 0.0
			for i, v := range cur {
				rowTotal := 0.0
				for j := 0; j < 10; j++ {
					rowTotal += mat[i][last[i]][j]
				}
				markov += mat[i][last[i]][v] / rowTotal
				positional += global[i][v] / float64(len(newest))
			}
			markov /= float64(positions)
			positional /= float64(positions)
			s := markov*0.4 + positional*0.3 + sumF[sumDigits(cur)]/float64(len(newest))*0.15 + uniqueF[len(uniqueDigits(cur))]/float64(len(newest))*0.1 + oddF[oddEvenKey(cur)]/float64(len(newest))*0.05
			out = append(out, scored{d: append([]int(nil), cur...), s: s})
			return
		}
		for _, d := range pool[p] {
			cur[p] = d
			walk(p + 1)
		}
	}
	walk(0)
	sort.Slice(out, func(i, j int) bool { return out[i].s > out[j].s })
	if top > len(out) {
		top = len(out)
	}
	return out[:top]
}

func reverse(in []data.DigitDraw) []data.DigitDraw {
	out := make([]data.DigitDraw, len(in))
	for i := range in {
		out[len(in)-1-i] = in[i]
	}
	return out
}

func topDigits(freq [10]float64, n int) []int {
	idx := make([]int, 10)
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(i, j int) bool { return freq[idx[i]] > freq[idx[j]] })
	return append([]int(nil), idx[:n]...)
}

func containsInt(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}
func sum3(d []int) int { return d[0] + d[1] + d[2] }
func sumDigits(d []int) int {
	s := 0
	for _, v := range d {
		s += v
	}
	return s
}
func min3(d []int) int {
	m := d[0]
	for _, v := range d[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
func max3(d []int) int {
	m := d[0]
	for _, v := range d[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
func oddEvenKey(d []int) int {
	o := 0
	for _, v := range d {
		o += v % 2
	}
	return o
}
func typeKey(d []int) int { return len(uniqueDigits(d)) }
func uniqueDigits(d []int) []int {
	seen := map[int]bool{}
	out := []int{}
	for _, v := range d {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}
func _unused() { _ = math.Abs }
