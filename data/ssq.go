// Package data 提供历史开奖 CSV 的读写（福彩3D + 双色球双彩种）。
package data

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

// SSQDraw 一期双色球开奖记录（6 红球 + 1 蓝球）
type SSQDraw struct {
	Issue                  string
	Date                   string
	R1, R2, R3, R4, R5, R6 int
	Blue                   int
}

// Reds 返回红球数组（按升序，即开奖顺序）
func (d SSQDraw) Reds() [6]int { return [6]int{d.R1, d.R2, d.R3, d.R4, d.R5, d.R6} }

// HasRed 判断某红球号码是否开出
func (d SSQDraw) HasRed(n int) bool {
	return d.R1 == n || d.R2 == n || d.R3 == n || d.R4 == n || d.R5 == n || d.R6 == n
}

// LoadSSQCSV 读取 ssq-history.csv，跳过表头与坏行（逐行读取，容忍脏数据）。
func LoadSSQCSV(path string) ([]SSQDraw, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	draws := make([]SSQDraw, 0, 4096)
	first := true
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || first {
			first = false
			continue
		}
		if len(rec) < 9 {
			continue
		}
		nums := make([]int, 0, 7)
		ok := true
		for i := 2; i < 9; i++ {
			n, e := strconv.Atoi(rec[i])
			if e != nil {
				ok = false
				break
			}
			nums = append(nums, n)
		}
		if !ok {
			continue
		}
		draws = append(draws, SSQDraw{
			Issue: rec[0], Date: rec[1],
			R1: nums[0], R2: nums[1], R3: nums[2], R4: nums[3], R5: nums[4], R6: nums[5],
			Blue: nums[6],
		})
	}
	return draws, nil
}

// AppendSSQCSV 追加一期；若期号已存在返回 0（不追加），否则追加并返回 1。
func AppendSSQCSV(path string, d SSQDraw) (int, error) {
	existing := map[string]bool{}
	if draws, err := LoadSSQCSV(path); err == nil {
		for _, dr := range draws {
			existing[dr.Issue] = true
		}
	}
	if existing[d.Issue] {
		return 0, nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if fi, err := f.Stat(); err == nil && fi.Size() == 0 {
		if _, err := f.WriteString("issue,date,r1,r2,r3,r4,r5,r6,blue\n"); err != nil {
			return 0, err
		}
	}
	w := csv.NewWriter(f)
	row := []string{d.Issue, d.Date,
		itoa2(d.R1), itoa2(d.R2), itoa2(d.R3), itoa2(d.R4), itoa2(d.R5), itoa2(d.R6),
		itoa2(d.Blue)}
	if err := w.Write(row); err != nil {
		return 0, err
	}
	w.Flush()
	return 1, w.Error()
}

// itoa2 双色球数字格式化为两位（01-33 / 01-16）
func itoa2(n int) string {
	if n < 10 {
		return "0" + fmt.Sprint(n)
	}
	return fmt.Sprint(n)
}

// LastSSQIssue 返回最新期号，空数据返回 ""。
func LastSSQIssue(draws []SSQDraw) string {
	if len(draws) == 0 {
		return ""
	}
	return draws[len(draws)-1].Issue
}
