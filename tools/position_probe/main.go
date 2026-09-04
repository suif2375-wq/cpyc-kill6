package main

import (
	"flag"
	"fmt"

	"fc3d-kill6/backtest"
	"fc3d-kill6/data"
	"fc3d-kill6/engine"
	"fc3d-kill6/engine/position"
)

func main() {
	path := flag.String("csv", "p5-history.csv", "数字型彩票 CSV")
	positions := flag.Int("positions", 5, "位数")
	window := flag.Int("window", 120, "近期窗口")
	top := flag.Int("top", 10, "推荐组数")
	flag.Parse()
	draws, err := data.LoadDigitCSV(*path, *positions)
	if err != nil {
		panic(err)
	}
	res := position.Backtest(draws, 2, *window)
	fmt.Printf("total=%d all=%.2f baseline=%.2f pred=%s\n", res.Total, res.AllRate, res.BaselineAll, position.FormatPrediction(res.Prediction))
	for _, st := range res.Stats {
		fmt.Printf("pos%d rate=%.2f baseline=%.2f model=%s n=%d\n", st.Position, st.Rate, st.Baseline, st.Model, st.N)
	}
	probeRecommendationCoverage(draws, *top)
	if *positions == 3 {
		legacy := make([]data.Draw, len(draws))
		for i, d := range draws {
			legacy[i] = data.Draw{Issue: d.Issue, Date: d.Date, B: d.Digits[0], S: d.Digits[1], G: d.Digits[2]}
		}
		lb := backtest.RunAll(legacy)
		fmt.Printf("legacy-v9 all6=%.2f pos=%.2f/%.2f/%.2f pred=%d,%d|%d,%d|%d,%d\n", lb.Meta.All6Pct, lb.Meta.AccH, lb.Meta.AccT, lb.Meta.AccO, lb.Pred.H, lb.Pred.H2, lb.Pred.T, lb.Pred.T2, lb.Pred.O, lb.Pred.O2)
	}
	if *positions == 5 {
		probeP5(draws)
		probeP5Adaptive(draws)
		probeP5Fixed(draws)
	}
}

func probeRecommendationCoverage(draws []data.DigitDraw, top int) {
	if len(draws) < 150 {
		return
	}
	checks := 100
	start := len(draws) - checks
	exact, two := 0, 0
	for t := start; t < len(draws); t++ {
		history := draws[:t]
		pred := position.Predict(history, 2, 120)
		recs := position.GenerateRecommendations(history, pred, top)
		for _, rec := range recs {
			matches := 0
			for p, d := range rec.Digits {
				if d == draws[t].Digits[p] {
					matches++
				}
			}
			if matches == len(draws[t].Digits) {
				exact++
				break
			}
			if matches >= 2 {
				two++
				break
			}
		}
	}
	fmt.Printf("recommend-top%d recent%d exact=%.2f%% two+pos=%.2f%%\n", top, checks, pct(exact, checks), pct(two, checks))
}

func probeP5(draws []data.DigitDraw) {
	// 把 V9 的三位公式作为排列5候选特征：每个位使用包含自身的局部三位上下文。
	ctx := [5][3]int{{0, 1, 2}, {0, 1, 2}, {1, 2, 3}, {2, 3, 4}, {2, 3, 4}}
	posIn := [5]int{0, 1, 1, 1, 2}
	good := make([]int, 5)
	n := len(draws) - 1
	for i := 1; i < len(draws); i++ {
		prev := draws[i-1].Digits
		act := draws[i].Digits
		for p := 0; p < 5; p++ {
			tr := [3]int{prev[ctx[p][0]], prev[ctx[p][1]], prev[ctx[p][2]]}
			var k1, k2 int
			switch posIn[p] {
			case 0:
				k1, k2 = engine.KillH(tr[0], tr[1], tr[2]), engine.KillH2(tr[0], tr[1], tr[2])
			case 1:
				k1, k2 = engine.KillT(tr[0], tr[1], tr[2]), engine.KillT2(tr[0], tr[1], tr[2])
			default:
				k1, k2 = engine.KillO(tr[0], tr[1], tr[2], nil, i), engine.KillO2(tr[0], tr[1], tr[2])
			}
			if act[p] != k1 && act[p] != k2 {
				good[p]++
			}
		}
	}
	fmt.Printf("p5-local-v9 pure pos=%.2f/%.2f/%.2f/%.2f/%.2f all=%.2f\n", pct(good[0], n), pct(good[1], n), pct(good[2], n), pct(good[3], n), pct(good[4], n), pctAll(draws, ctx, posIn))
	searchP5Contexts(draws)
}

func searchP5Contexts(draws []data.DigitDraw) {
	for target := 0; target < 5; target++ {
		bestRate, bestA, bestB, bestC, bestSlot := 0.0, 0, 0, 0, 0
		for a := 0; a < 5; a++ {
			for b := a + 1; b < 5; b++ {
				for c := b + 1; c < 5; c++ {
					if target != a && target != b && target != c {
						continue
					}
					for slot := 0; slot < 3; slot++ {
						good, total := 0, 0
						for i := 31; i < len(draws); i++ {
							tr := [3]int{draws[i-1].Digits[a], draws[i-1].Digits[b], draws[i-1].Digits[c]}
							var k1, k2 int
							switch slot {
							case 0:
								k1, k2 = engine.KillH(tr[0], tr[1], tr[2]), engine.KillH2(tr[0], tr[1], tr[2])
							case 1:
								k1, k2 = engine.KillT(tr[0], tr[1], tr[2]), engine.KillT2(tr[0], tr[1], tr[2])
							default:
								k1, k2 = engine.KillO(tr[0], tr[1], tr[2], nil, i), engine.KillO2(tr[0], tr[1], tr[2])
							}
							total++
							if draws[i].Digits[target] != k1 && draws[i].Digits[target] != k2 {
								good++
							}
						}
						rate := pct(good, total)
						if rate > bestRate {
							bestRate, bestA, bestB, bestC, bestSlot = rate, a, b, c, slot
						}
					}
				}
			}
		}
		fmt.Printf("p5-search target%d ctx=%d,%d,%d slot=%d rate=%.2f\n", target+1, bestA, bestB, bestC, bestSlot, bestRate)
	}
}

type p5Candidate struct{ a, b, c, slot int }

func probeP5Adaptive(draws []data.DigitDraw) {
	if len(draws) < 160 {
		return
	}
	cands := make([][]p5Candidate, 5)
	for target := 0; target < 5; target++ {
		for a := 0; a < 5; a++ {
			for b := a + 1; b < 5; b++ {
				for c := b + 1; c < 5; c++ {
					if target != a && target != b && target != c {
						continue
					}
					for slot := 0; slot < 3; slot++ {
						cands[target] = append(cands[target], p5Candidate{a, b, c, slot})
					}
				}
			}
		}
	}
	good, total := 0, 0
	posGood := make([]int, 5)
	for t := 31; t < len(draws); t++ {
		history := draws[:t]
		for p := 0; p < 5; p++ {
			best := chooseP5Candidate(history, p, cands[p])
			k1, k2 := applyP5Candidate(history[len(history)-1].Digits, best)
			if draws[t].Digits[p] != k1 && draws[t].Digits[p] != k2 {
				good++
				posGood[p]++
			}
			total++
		}
	}
	all := 0
	for t := 31; t < len(draws); t++ {
		ok := true
		for p := 0; p < 5; p++ {
			best := chooseP5Candidate(draws[:t], p, cands[p])
			k1, k2 := applyP5Candidate(draws[t-1].Digits, best)
			if draws[t].Digits[p] == k1 || draws[t].Digits[p] == k2 {
				ok = false
				break
			}
		}
		if ok {
			all++
		}
	}
	fmt.Printf("p5-adaptive candidates pos=%.2f/%.2f/%.2f/%.2f/%.2f all=%.2f\n", pct(posGood[0], len(draws)-31), pct(posGood[1], len(draws)-31), pct(posGood[2], len(draws)-31), pct(posGood[3], len(draws)-31), pct(posGood[4], len(draws)-31), pct(all, len(draws)-31))
}

func probeP5Fixed(draws []data.DigitDraw) {
	fixed := []p5Candidate{{0, 2, 3, 0}, {1, 2, 3, 1}, {0, 2, 3, 1}, {2, 3, 4, 1}, {1, 3, 4, 1}}
	all, total := 0, 0
	good := make([]int, 5)
	for i := 31; i < len(draws); i++ {
		ok := true
		for p, cand := range fixed {
			k1, k2 := applyP5Candidate(draws[i-1].Digits, cand)
			if draws[i].Digits[p] == k1 || draws[i].Digits[p] == k2 {
				ok = false
			} else {
				good[p]++
			}
		}
		if ok {
			all++
		}
		total++
	}
	fmt.Printf("p5-fixed-search pos=%.2f/%.2f/%.2f/%.2f/%.2f all=%.2f\n", pct(good[0], total), pct(good[1], total), pct(good[2], total), pct(good[3], total), pct(good[4], total), pct(all, total))
	probeP5TrainHoldout(draws)
}

func probeP5TrainHoldout(draws []data.DigitDraw) {
	cut := len(draws) * 7 / 10
	if cut < 160 {
		return
	}
	allCands := make([][]p5Candidate, 5)
	for target := 0; target < 5; target++ {
		for a := 0; a < 5; a++ {
			for b := a + 1; b < 5; b++ {
				for c := b + 1; c < 5; c++ {
					if target != a && target != b && target != c {
						continue
					}
					for slot := 0; slot < 3; slot++ {
						allCands[target] = append(allCands[target], p5Candidate{a, b, c, slot})
					}
				}
			}
		}
	}
	fixed := make([]p5Candidate, 5)
	for p := 0; p < 5; p++ {
		fixed[p] = chooseP5Candidate(draws[:cut], p, allCands[p])
	}
	good, total, all := make([]int, 5), 0, 0
	defaultMap := []p5Candidate{{0, 1, 2, 0}, {0, 1, 2, 1}, {1, 2, 3, 1}, {2, 3, 4, 1}, {2, 3, 4, 2}}
	defaultGood, defaultAll := make([]int, 5), 0
	for i := cut; i < len(draws); i++ {
		ok := true
		defaultOK := true
		for p, cand := range fixed {
			k1, k2 := applyP5Candidate(draws[i-1].Digits, cand)
			if draws[i].Digits[p] == k1 || draws[i].Digits[p] == k2 {
				ok = false
			} else {
				good[p]++
			}
			d1, d2 := applyP5Candidate(draws[i-1].Digits, defaultMap[p])
			if draws[i].Digits[p] == d1 || draws[i].Digits[p] == d2 {
				defaultOK = false
			} else {
				defaultGood[p]++
			}
		}
		if ok {
			all++
		}
		if defaultOK {
			defaultAll++
		}
		total++
	}
	fmt.Printf("p5-train70-holdout30 pos=%.2f/%.2f/%.2f/%.2f/%.2f all=%.2f ctx=%v default=%.2f/%.2f/%.2f/%.2f/%.2f all=%.2f\n", pct(good[0], total), pct(good[1], total), pct(good[2], total), pct(good[3], total), pct(good[4], total), pct(all, total), fixed, pct(defaultGood[0], total), pct(defaultGood[1], total), pct(defaultGood[2], total), pct(defaultGood[3], total), pct(defaultGood[4], total), pct(defaultAll, total))
}

func chooseP5Candidate(history []data.DigitDraw, target int, cands []p5Candidate) p5Candidate {
	start := len(history) - 120
	if start < 31 {
		start = 31
	}
	best := cands[0]
	bestScore := -1
	for _, cand := range cands {
		score := 0
		for t := start; t < len(history); t++ {
			k1, k2 := applyP5Candidate(history[t-1].Digits, cand)
			if history[t].Digits[target] != k1 && history[t].Digits[target] != k2 {
				score++
			}
		}
		if score > bestScore {
			bestScore, best = score, cand
		}
	}
	return best
}

func applyP5Candidate(d []int, cand p5Candidate) (int, int) {
	b, s, g := d[cand.a], d[cand.b], d[cand.c]
	switch cand.slot {
	case 0:
		return engine.KillH(b, s, g), engine.KillH2(b, s, g)
	case 1:
		return engine.KillT(b, s, g), engine.KillT2(b, s, g)
	default:
		return engine.KillO(b, s, g, nil, 0), engine.KillO2(b, s, g)
	}
}

func pct(hit, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(hit) / float64(n) * 100
}

func pctAll(draws []data.DigitDraw, ctx [5][3]int, posIn [5]int) float64 {
	good, n := 0, 0
	for i := 1; i < len(draws); i++ {
		if i < 30 {
			continue
		}
		n++
		prev, act := draws[i-1].Digits, draws[i].Digits
		ok := true
		for p := 0; p < 5; p++ {
			tr := [3]int{prev[ctx[p][0]], prev[ctx[p][1]], prev[ctx[p][2]]}
			var k1, k2 int
			switch posIn[p] {
			case 0:
				k1, k2 = engine.KillH(tr[0], tr[1], tr[2]), engine.KillH2(tr[0], tr[1], tr[2])
			case 1:
				k1, k2 = engine.KillT(tr[0], tr[1], tr[2]), engine.KillT2(tr[0], tr[1], tr[2])
			default:
				k1, k2 = engine.KillO(tr[0], tr[1], tr[2], nil, i), engine.KillO2(tr[0], tr[1], tr[2])
			}
			if act[p] == k1 || act[p] == k2 {
				ok = false
				break
			}
		}
		if ok {
			good++
		}
	}
	return pct(good, n)
}
