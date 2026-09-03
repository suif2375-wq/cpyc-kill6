// tools/ssq_recent — 双色球最近窗口策略表现诊断（一次性工具）。
//
// 对比：最近 50/100 期，冷/热/遗漏策略在各自窗口下的杀中率 vs 随机基线。
// 目的：判断"最近窗口是否有可优化的策略空间"，供产品决策。
//
// 用法: go run ./tools/ssq_recent -csv ssq-history.csv
package main

import (
	"flag"
	"fmt"
	"os"

	"fc3d-kill6/data"
	"fc3d-kill6/engine/ssq"
)

func main() {
	csvPath := flag.String("csv", "ssq-history.csv", "双色球 CSV 路径")
	flag.Parse()

	draws, err := data.LoadSSQCSV(*csvPath)
	if err != nil || len(draws) < 300 {
		fmt.Printf("❌ 读取失败: %v (%d 期)\n", err, len(draws))
		os.Exit(1)
	}

	redBase := map[int]float64{6: comb(27, 6) / comb(33, 6) * 100, 8: comb(25, 6) / comb(33, 6) * 100}
	blueBase := func(m int) float64 { return float64(16-m) / 16 * 100 }
	strats := []ssq.Strategy{ssq.StrategyCold, ssq.StrategyHot, ssq.StrategyMiss}

	for _, recent := range []int{50, 100} {
		start := len(draws) - recent
		fmt.Printf("══ 最近 %d 期（%s ~ %s）══\n", recent, draws[start].Issue, draws[len(draws)-1].Issue)
		fmt.Printf("%-8s %-6s %8s %8s %8s %8s\n", "策略", "窗口", "杀红%", "杀蓝%", "全中%", "基线全中%")
		for _, s := range strats {
			for _, w := range []int{20, 50, 100} {
				rhit, bhit, allhit, n := 0, 0, 0, 0
				for t := start; t < len(draws); t++ {
					if t < w {
						continue
					}
					win := draws[t-w : t]
					kr := ssq.KillReds(win, 6, 0, s)
					kb := ssq.KillBlues(win, 3, 0, s)
					d := draws[t]
					redOK, blueOK := true, true
					for _, r := range d.Reds() {
						if contains(kr, r) {
							redOK = false
							break
						}
					}
					if contains(kb, d.Blue) {
						blueOK = false
					}
					if redOK {
						rhit++
					}
					if blueOK {
						bhit++
					}
					if redOK && blueOK {
						allhit++
					}
					n++
				}
				if n == 0 {
					continue
				}
				fmt.Printf("%-8s %-6d %7.1f%% %7.1f%% %7.1f%% %8.1f%%\n",
					s.StrName(), w, 100*float64(rhit)/float64(n), 100*float64(bhit)/float64(n),
					100*float64(allhit)/float64(n), redBase[6]*blueBase(3)/100)
			}
		}
		fmt.Println()
	}

	fmt.Printf("参考基线: 杀6红 %.1f%% | 杀3蓝 %.1f%% | 全中 %.1f%%\n",
		redBase[6], blueBase(3), redBase[6]*blueBase(3)/100)
	fmt.Printf("福彩3D 同口径参考: 近100期 6杀全中 81%%（基线 51.2%%）—— 空间结构不同，不可直接对比\n")
}

func contains(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

func comb(n, k int) float64 {
	r := 1.0
	for i := 0; i < k; i++ {
		r *= float64(n - i)
		r /= float64(i + 1)
	}
	return r
}
