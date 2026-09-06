// position_eval 对排列3/排列5杀号方案做严格时间顺序评估。
package main

import (
	"fmt"
	"math"

	"fc3d-kill6/data"
	"fc3d-kill6/engine"
)

type candidate struct{ a, b, c, slot int }

type result struct {
	name string
	hits []bool
	pos  [][]bool
}

func main() {
	p3, err := data.LoadDigitCSV("p3-history.csv", 3)
	if err != nil {
		panic(err)
	}
	p5, err := data.LoadDigitCSV("p5-history.csv", 5)
	if err != nil {
		panic(err)
	}
	checkPrefix(p3, p5)
	checkPredictionPrefix(p3, p5)

	p3Stateful, p3Pure := evalP3(p3)
	printResult(p3Stateful)
	printResult(p3Pure)
	for _, window := range []int{60, 120, 240} {
		printResult(evalP3RareFallback(p3, window))
	}

	currentTail := [2]candidate{{2, 3, 4, 1}, {2, 3, 4, 2}}
	trainedFifth := [2]candidate{{2, 3, 4, 1}, {0, 2, 4, 1}}
	fullSearchTail := [2]candidate{{2, 3, 4, 1}, {1, 3, 4, 1}}
	holdoutTail := [2]candidate{{1, 3, 4, 2}, {0, 2, 4, 1}}
	for _, r := range []result{
		evalP5(p5, true, currentTail, "P5 当前: 状态机前三位 + 当前尾部"),
		evalP5(p5, false, currentTail, "P5 纯公式前三位 + 当前尾部"),
		evalP5(p5, true, trainedFifth, "P5 状态机前三位 + 训练集第五位"),
		evalP5(p5, false, trainedFifth, "P5 纯公式前三位 + 训练集第五位"),
		evalP5(p5, true, fullSearchTail, "P5 状态机前三位 + 全量搜索尾部"),
		evalP5(p5, false, fullSearchTail, "P5 纯公式前三位 + 全量搜索尾部"),
		evalP5(p5, true, holdoutTail, "P5 状态机前三位 + 70%训练尾部"),
		evalP5(p5, false, holdoutTail, "P5 纯公式前三位 + 70%训练尾部"),
	} {
		printResult(r)
	}
	for _, window := range []int{120, 240, 500} {
		for _, margin := range []int{0, 2, 4} {
			printResult(evalP5AdaptiveTail(p5, window, margin))
		}
	}
	for _, window := range []int{60, 120, 240} {
		printResult(evalP5RareFallback(p5, window))
	}
}

func checkPredictionPrefix(p3, p5 []data.DigitDraw) {
	type triple [3][2]int
	build := func(draws []data.DigitDraw) map[string]triple {
		out := make(map[string]triple, len(draws)-1)
		st := engine.NewState()
		for t := 1; t < len(draws); t++ {
			prev, actual := draws[t-1].Digits, draws[t].Digits
			h, s, g, h2, s2, g2 := st.Next(prev[0], prev[1], prev[2], actual[2])
			out[draws[t].Issue] = triple{
				pair(h, h2, 0, prev[:3]),
				pair(s, s2, 1, prev[:3]),
				pair(g, g2, 2, prev[:3]),
			}
		}
		return out
	}
	a, b := build(p3), build(p5)
	mismatch, recentMismatch := 0, 0
	for issue, p3Kills := range a {
		if issue < "2004021" {
			continue
		}
		if p5Kills, ok := b[issue]; ok && p3Kills != p5Kills {
			mismatch++
			if issue >= "2026001" {
				recentMismatch++
			}
		}
	}
	fmt.Printf("P3/P5前三位杀码一致性: 2004021后不一致=%d 2026年不一致=%d 当前一致=%v\n", mismatch, recentMismatch, a[p3[len(p3)-1].Issue] == b[p5[len(p5)-1].Issue])
}

func checkPrefix(p3, p5 []data.DigitDraw) {
	byIssue := make(map[string]data.DigitDraw, len(p3))
	for _, d := range p3 {
		byIssue[d.Issue] = d
	}
	matched, earlyIndependent, sharedMismatch := 0, 0, 0
	for _, d5 := range p5 {
		d3, ok := byIssue[d5.Issue]
		if !ok {
			continue
		}
		matched++
		if d3.Date != d5.Date || len(d5.Digits) < 3 || d3.Digits[0] != d5.Digits[0] || d3.Digits[1] != d5.Digits[1] || d3.Digits[2] != d5.Digits[2] {
			if d5.Issue < "2004021" {
				earlyIndependent++
			} else {
				sharedMismatch++
			}
		}
	}
	fmt.Printf("P3/P5历史一致性: 共同期号=%d 共同摇奖前独立期=%d 第2004021期后不一致=%d\n", matched, earlyIndependent, sharedMismatch)
}

func evalP3(draws []data.DigitDraw) (result, result) {
	stateful := newResult("P3 当前状态机", len(draws)-1, 3)
	pure := newResult("P3 纯V9公式", len(draws)-1, 3)
	st := engine.NewState()
	for t := 1; t < len(draws); t++ {
		prev, actual := draws[t-1].Digits, draws[t].Digits
		h, s, g, h2, s2, g2 := st.Next(prev[0], prev[1], prev[2], actual[2])
		statePairs := [3][2]int{
			pair(h, h2, 0, prev), pair(s, s2, 1, prev), pair(g, g2, 2, prev),
		}
		purePairs := [3][2]int{
			pair(engine.KillH(prev[0], prev[1], prev[2]), engine.KillH2(prev[0], prev[1], prev[2]), 0, prev),
			pair(engine.KillT(prev[0], prev[1], prev[2]), engine.KillT2(prev[0], prev[1], prev[2]), 1, prev),
			pair(engine.KillO(prev[0], prev[1], prev[2], nil, t), engine.KillO2(prev[0], prev[1], prev[2]), 2, prev),
		}
		record(&stateful, t-1, actual, statePairs[:])
		record(&pure, t-1, actual, purePairs[:])
	}
	return stateful, pure
}

func evalP3RareFallback(draws []data.DigitDraw, window int) result {
	r := newResult(fmt.Sprintf("P3 状态机 + 重码低概率回退 w=%d", window), len(draws)-1, 3)
	st := engine.NewState()
	for t := 1; t < len(draws); t++ {
		prev, actual := draws[t-1].Digits, draws[t].Digits
		h, s, g, h2, s2, g2 := st.Next(prev[0], prev[1], prev[2], actual[2])
		pairs := [3][2]int{
			pairRare(h, h2, draws[:t], 0, window),
			pairRare(s, s2, draws[:t], 1, window),
			pairRare(g, g2, draws[:t], 2, window),
		}
		record(&r, t-1, actual, pairs[:])
	}
	return r
}

func evalP5(draws []data.DigitDraw, statefulPrefix bool, tail [2]candidate, name string) result {
	r := newResult(name, len(draws)-1, 5)
	st := engine.NewState()
	for t := 1; t < len(draws); t++ {
		prev, actual := draws[t-1].Digits, draws[t].Digits
		h, s, g, h2, s2, g2 := st.Next(prev[0], prev[1], prev[2], actual[2])
		pairs := make([][2]int, 5)
		if statefulPrefix {
			pairs[0], pairs[1], pairs[2] = pair(h, h2, 0, prev[:3]), pair(s, s2, 1, prev[:3]), pair(g, g2, 2, prev[:3])
		} else {
			pairs[0] = pair(engine.KillH(prev[0], prev[1], prev[2]), engine.KillH2(prev[0], prev[1], prev[2]), 0, prev[:3])
			pairs[1] = pair(engine.KillT(prev[0], prev[1], prev[2]), engine.KillT2(prev[0], prev[1], prev[2]), 1, prev[:3])
			pairs[2] = pair(engine.KillO(prev[0], prev[1], prev[2], nil, t), engine.KillO2(prev[0], prev[1], prev[2]), 2, prev[:3])
		}
		pairs[3] = apply(prev, tail[0])
		pairs[4] = apply(prev, tail[1])
		record(&r, t-1, actual, pairs)
	}
	return r
}

func evalP5RareFallback(draws []data.DigitDraw, window int) result {
	r := newResult(fmt.Sprintf("P5 当前映射 + 重码低概率回退 w=%d", window), len(draws)-1, 5)
	st := engine.NewState()
	tail := [2]candidate{{2, 3, 4, 1}, {2, 3, 4, 2}}
	for t := 1; t < len(draws); t++ {
		prev, actual := draws[t-1].Digits, draws[t].Digits
		h, s, g, h2, s2, g2 := st.Next(prev[0], prev[1], prev[2], actual[2])
		pairs := make([][2]int, 5)
		pairs[0] = pairRare(h, h2, draws[:t], 0, window)
		pairs[1] = pairRare(s, s2, draws[:t], 1, window)
		pairs[2] = pairRare(g, g2, draws[:t], 2, window)
		for p := 3; p < 5; p++ {
			c := tail[p-3]
			b, s0, g0 := prev[c.a], prev[c.b], prev[c.c]
			var k1, k2 int
			switch c.slot {
			case 0:
				k1, k2 = engine.KillH(b, s0, g0), engine.KillH2(b, s0, g0)
			case 1:
				k1, k2 = engine.KillT(b, s0, g0), engine.KillT2(b, s0, g0)
			default:
				k1, k2 = engine.KillO(b, s0, g0, nil, 0), engine.KillO2(b, s0, g0)
			}
			pairs[p] = pairRare(k1, k2, draws[:t], p, window)
		}
		record(&r, t-1, actual, pairs)
	}
	return r
}

func pairRare(k1, k2 int, history []data.DigitDraw, pos, window int) [2]int {
	if k1 != k2 {
		return [2]int{k1, k2}
	}
	freq := make([]float64, 10)
	trans := make([]float64, 10)
	for d := 0; d < 10; d++ {
		freq[d], trans[d] = 1, 1
	}
	start := len(history) - window
	if start < 0 {
		start = 0
	}
	for i := start; i < len(history); i++ {
		age := len(history) - 1 - i
		freq[history[i].Digits[pos]] += math.Pow(0.96, float64(age))
	}
	transSamples := 0
	if len(history) > 1 {
		last := history[len(history)-1].Digits[pos]
		transStart := start
		if transStart < 1 {
			transStart = 1
		}
		for i := transStart; i < len(history); i++ {
			if history[i-1].Digits[pos] == last {
				age := len(history) - 1 - i
				trans[history[i].Digits[pos]] += math.Pow(0.90, float64(age))
				transSamples++
			}
		}
	}
	freqTotal, transTotal := 0.0, 0.0
	for d := 0; d < 10; d++ {
		freqTotal += freq[d]
		transTotal += trans[d]
	}
	best, bestScore := -1, math.MaxFloat64
	for d := 0; d < 10; d++ {
		if d == k1 {
			continue
		}
		score := freq[d] / freqTotal
		if transSamples > 0 {
			score = 0.65*score + 0.35*trans[d]/transTotal
		}
		if score < bestScore {
			best, bestScore = d, score
		}
	}
	return [2]int{k1, best}
}

func evalP5AdaptiveTail(draws []data.DigitDraw, window, margin int) result {
	name := fmt.Sprintf("P5 状态机前三位 + 尾部滚选 w=%d margin=%d", window, margin)
	r := newResult(name, len(draws)-1, 5)
	st := engine.NewState()
	defaults := [2]candidate{{2, 3, 4, 1}, {2, 3, 4, 2}}
	candidates := [2][]candidate{candidatePool(3), candidatePool(4)}
	for t := 1; t < len(draws); t++ {
		prev, actual := draws[t-1].Digits, draws[t].Digits
		h, s, g, h2, s2, g2 := st.Next(prev[0], prev[1], prev[2], actual[2])
		pairs := make([][2]int, 5)
		pairs[0], pairs[1], pairs[2] = pair(h, h2, 0, prev[:3]), pair(s, s2, 1, prev[:3]), pair(g, g2, 2, prev[:3])
		for tailPos := 0; tailPos < 2; tailPos++ {
			chosen := defaults[tailPos]
			if t >= 60 {
				chosen = chooseCandidate(draws[:t], tailPos+3, candidates[tailPos], defaults[tailPos], window, margin)
			}
			pairs[tailPos+3] = apply(prev, chosen)
		}
		record(&r, t-1, actual, pairs)
	}
	return r
}

func candidatePool(target int) []candidate {
	out := make([]candidate, 0, 18)
	for a := 0; a < 5; a++ {
		for b := a + 1; b < 5; b++ {
			for c := b + 1; c < 5; c++ {
				if target != a && target != b && target != c {
					continue
				}
				for slot := 0; slot < 3; slot++ {
					out = append(out, candidate{a, b, c, slot})
				}
			}
		}
	}
	return out
}

func chooseCandidate(history []data.DigitDraw, target int, candidates []candidate, fallback candidate, window, margin int) candidate {
	start := len(history) - window
	if start < 1 {
		start = 1
	}
	score := func(c candidate) int {
		hit := 0
		for t := start; t < len(history); t++ {
			ks := apply(history[t-1].Digits, c)
			if history[t].Digits[target] != ks[0] && history[t].Digits[target] != ks[1] {
				hit++
			}
		}
		return hit
	}
	best, bestScore := fallback, score(fallback)
	fallbackScore := bestScore
	for _, c := range candidates {
		if s := score(c); s > bestScore {
			best, bestScore = c, s
		}
	}
	if bestScore < fallbackScore+margin {
		return fallback
	}
	return best
}

func apply(d []int, c candidate) [2]int {
	b, s, g := d[c.a], d[c.b], d[c.c]
	var k1, k2 int
	switch c.slot {
	case 0:
		k1, k2 = engine.KillH(b, s, g), engine.KillH2(b, s, g)
	case 1:
		k1, k2 = engine.KillT(b, s, g), engine.KillT2(b, s, g)
	default:
		k1, k2 = engine.KillO(b, s, g, nil, 0), engine.KillO2(b, s, g)
	}
	return pair(k1, k2, c.slot, []int{b, s, g})
}

func pair(k1, k2, slot int, d []int) [2]int {
	if k1 != k2 {
		return [2]int{k1, k2}
	}
	b, s, g := d[0], d[1], d[2]
	alts := [][]int{
		{(k1 + 1) % 10, engine.KillT(b, s, g), engine.KillO(b, s, g, nil, 0), engine.KillH2(b, s, g), (k1 + 3) % 10},
		{(k1 + 1) % 10, engine.KillH(b, s, g), engine.KillO(b, s, g, nil, 0), engine.KillT2(b, s, g), (k1 + 3) % 10},
		{(k1 + 1) % 10, engine.KillH(b, s, g), engine.KillT(b, s, g), engine.KillO2(b, s, g), (k1 + 3) % 10},
	}[slot]
	for _, alt := range alts {
		if alt != k1 {
			return [2]int{k1, alt}
		}
	}
	return [2]int{k1, (k1 + 1) % 10}
}

func newResult(name string, n, positions int) result {
	r := result{name: name, hits: make([]bool, n), pos: make([][]bool, positions)}
	for p := range r.pos {
		r.pos[p] = make([]bool, n)
	}
	return r
}

func record(r *result, idx int, actual []int, pairs [][2]int) {
	all := true
	for p, ks := range pairs {
		ok := actual[p] != ks[0] && actual[p] != ks[1]
		r.pos[p][idx] = ok
		all = all && ok
	}
	r.hits[idx] = all
}

func printResult(r result) {
	fmt.Printf("\n%s\n", r.name)
	for _, w := range []int{0, 2000, 1000, 500, 200, 100} {
		start := 29
		label := "全量"
		if w > 0 && len(r.hits)-w > start {
			start = len(r.hits) - w
			label = fmt.Sprintf("近%d", w)
		}
		fmt.Printf("  %-6s all=%6.2f%%", label, rate(r.hits[start:]))
		for p := range r.pos {
			fmt.Printf(" p%d=%5.2f", p+1, rate(r.pos[p][start:]))
		}
		fmt.Println()
	}
	cut := int(math.Floor(float64(len(r.hits)) * 0.7))
	fmt.Printf("  后30%%  all=%6.2f%%", rate(r.hits[cut:]))
	for p := range r.pos {
		fmt.Printf(" p%d=%5.2f", p+1, rate(r.pos[p][cut:]))
	}
	fmt.Println()
}

func rate(a []bool) float64 {
	if len(a) == 0 {
		return 0
	}
	n := 0
	for _, ok := range a {
		if ok {
			n++
		}
	}
	return float64(n) / float64(len(a)) * 100
}
