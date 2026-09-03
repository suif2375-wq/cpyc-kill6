// tools/import_ssq — 一次性导入双色球全量历史（17500 ssq.TXT）→ ssq-history.csv。
//
// 用法:
//
//	curl -sL -A "Mozilla/5.0" https://www.17500.cn/getData/ssq.TXT -o /tmp/ssq_full.txt
//	go run ./tools/import_ssq -in /tmp/ssq_full.txt -out ssq-history.csv
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"fc3d-kill6/data"
)

func main() {
	inPath := flag.String("in", "ssq_full.txt", "17500 ssq TXT 路径")
	outPath := flag.String("out", "ssq-history.csv", "输出 CSV 路径")
	flag.Parse()

	f, err := os.Open(*inPath)
	if err != nil {
		fmt.Printf("❌ 打开 %s 失败: %v\n", *inPath, err)
		os.Exit(1)
	}
	defer f.Close()

	issueRe := regexp.MustCompile(`^20\d{5}$`)
	count := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 || !issueRe.MatchString(fields[0]) {
			continue
		}
		nums := make([]int, 0, 7)
		ok := true
		for i := 2; i < 9; i++ {
			n, e := strconv.Atoi(fields[i])
			if e != nil {
				ok = false
				break
			}
			nums = append(nums, n)
		}
		if !ok || nums[6] < 1 || nums[6] > 16 {
			continue
		}
		draw := data.SSQDraw{
			Issue: fields[0], Date: fields[1],
			R1: nums[0], R2: nums[1], R3: nums[2], R4: nums[3], R5: nums[4], R6: nums[5],
			Blue: nums[6],
		}
		if n, err := data.AppendSSQCSV(*outPath, draw); err != nil {
			fmt.Printf("⚠️ %s 写入失败: %v\n", draw.Issue, err)
		} else if n == 1 {
			count++
		}
	}
	if err := sc.Err(); err != nil {
		fmt.Printf("❌ 读取失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ ssq-history.csv 导入完成: %d 期\n", count)
}
