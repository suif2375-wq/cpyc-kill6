// tools/ssq_probe — 双色球杀号策略回测对比（一次性诊断工具）。
//
// 对每种 策略×窗口 组合，滚动回测"杀 N 红 + 杀 M 蓝"的命中率 vs 随机基线，
// 输出表格，据此决定最终采用的策略与杀号数量。
//
// 用法: go run ./tools/ssq_probe -csv ssq-history.csv
package main

import (
	"flag"
	"fmt"
	"math"
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

	// 组合空间
	nReds := []int{6, 8}
	nBlues := []int{3, 5}
	windows := []int{20, 50, 100}
	strategies := []ssq.Strategy{ssq.StrategyCold, ssq.StrategyHot, ssq.StrategyMiss}

	// 基线
	redBase := func(n int) float64 { return comb(33-n, 6) / comb(33, 6) * 100 }
	blueBase := func(m int) float64 { return float64(16-m) / 16 * 100 }

	fmt.Printf("双色球全量 %d 期 · 基线参考: 红基线=杀%d→%.1f%%/杀%d→%.1f%%; 蓝基线=杀%d→%.1f%%/杀%d→%.1f%%\n\n",
		len(draws), nReds[0], redBase(nReds[0]), nReds[1], redBase(nReds[1]), nBlues[0], blueBase(nBlues[0]), nBlues[1], blueBase(nBlues[1]))

	for _, nr := range nReds {
		for _, nb := range nBlues {
			rb, bb := redBase(nr), blueBase(nb)
			allBase := rb * bb / 100
			fmt.Printf("== 杀 %d 红 + 杀 %d 蓝 (基线 红%.1f%% 蓝%.1f%% 全中%.1f%%) ==\n", nr, nb, rb, bb, allBase)
			fmt.Printf("%-8s %-8s %8s %8s %8s\n", "策略", "窗口", "红全中%", "蓝全中%", "全中%")
			for _, s := range strategies {
				for _, w := range windows {
					rhit, bhit, allhit, n := 0, 0, 0, 0
					for t := w; t < len(draws); t++ {
						win := draws[t-w : t]
						kr := ssq.KillReds(win, nr, 0, s)
						kb := ssq.KillBlues(win, nb, 0, s)
						d := draws[t]
						redOK := true
						for _, r := range d.Reds() {
							if contains(kr, r) {
								redOK = false
								break
							}
						}
						blueOK := !contains(kb, d.Blue)
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
					fmt.Printf("%-8s %-8d %7.1f%% %7.1f%% %7.1f%%\n",
						s.StrName(), w, 100*float64(rhit)/float64(n), 100*float64(bhit)/float64(n), 100*float64(allhit)/float64(n))
				}
			}
			fmt.Println()
		}
	}
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
	if k > n {
		return 0
	}
	r := 1.0
	for i := 0; i < k; i++ {
		r *= float64(n - i)
		r /= float64(i + 1)
	}
	return math.Round(r)
}
