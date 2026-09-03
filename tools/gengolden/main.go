// tools/gengolden — 生成 golden.json 引擎基准（Go 版，替代原 Python 脚本）。
//
// 用法: go run ./tools/gengolden -csv fc3d-history.csv -out golden.json
// 输出: 全量逐期 6 杀码 + 1000 组 (0-9)³ 纯函数穷举。
// 仅开发期使用，不入库核心链路。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"fc3d-kill6/data"
	"fc3d-kill6/engine"
)

type goldenRow struct {
	Issue string `json:"issue"`
	HK    int    `json:"hK"`
	TK    int    `json:"tK"`
	OK    int    `json:"oK"`
	HK2   int    `json:"hK2"`
	TK2   int    `json:"tK2"`
	OK2   int    `json:"oK2"`
}

type goldenExact struct {
	B  int `json:"b"`
	S  int `json:"s"`
	G  int `json:"g"`
	H  int `json:"h"`
	T  int `json:"t"`
	O  int `json:"o"`
	H2 int `json:"h2"`
	T2 int `json:"t2"`
	O2 int `json:"o2"`
}

type goldenFile struct {
	Source     string        `json:"source"`
	Total      int           `json:"total"`
	Rows       []goldenRow   `json:"rows"`
	Exhaustive []goldenExact `json:"exhaustive"`
}

func main() {
	csvPath := flag.String("csv", "fc3d-history.csv", "历史开奖 CSV 路径")
	outPath := flag.String("out", "golden.json", "输出 golden JSON 路径")
	flag.Parse()

	draws, err := data.LoadCSV(*csvPath)
	if err != nil || len(draws) < 100 {
		fmt.Printf("❌ 读取 CSV 失败: %v (%d 期)\n", err, len(draws))
		os.Exit(1)
	}

	// 全量逐期状态机（与引擎测试 TestGoldenRows 同逻辑）
	st := engine.NewState()
	rows := make([]goldenRow, 0, len(draws)-1)
	for i := 1; i < len(draws); i++ {
		p := draws[i-1]
		d := draws[i]
		hk, tk, ok, hk2, tk2, ok2 := st.Next(p.B, p.S, p.G, d.G)
		rows = append(rows, goldenRow{Issue: d.Issue, HK: hk, TK: tk, OK: ok, HK2: hk2, TK2: tk2, OK2: ok2})
	}

	// 1000 组全量穷举（纯函数，无状态）
	ex := make([]goldenExact, 0, 1000)
	for b := 0; b < 10; b++ {
		for s := 0; s < 10; s++ {
			for g := 0; g < 10; g++ {
				ex = append(ex, goldenExact{
					B: b, S: s, G: g,
					H:  engine.KillH(b, s, g),
					T:  engine.KillT(b, s, g),
					O:  engine.KillO(b, s, g, nil, 0),
					H2: engine.KillH2(b, s, g),
					T2: engine.KillT2(b, s, g),
					O2: engine.KillO2(b, s, g),
				})
			}
		}
	}

	out := goldenFile{
		Source:     "fc3d-kill6 V9.3 引擎基准 (tools/gengolden)",
		Total:      len(rows),
		Rows:       rows,
		Exhaustive: ex,
	}
	raw, err := json.Marshal(out)
	if err != nil {
		fmt.Printf("❌ 序列化失败: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, raw, 0o644); err != nil {
		fmt.Printf("❌ 写入失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("golden.json 生成完成: %d 期逐期 + %d 组全量穷举 (1000组合)\n", len(rows), len(ex))
}
