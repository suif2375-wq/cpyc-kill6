// Package data 提供历史开奖 CSV 的读写。
package data

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"
)

// Draw 一期开奖记录
type Draw struct {
	Issue   string
	Date    string
	B, S, G int
}

// LoadCSV 读取 fc3d-history.csv，跳过表头与坏行（逐行读取，容忍脏数据）。
func LoadCSV(path string) ([]Draw, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // 允许行长度不一致，坏行跳过而非整体失败
	draws := make([]Draw, 0, 4096)
	first := true
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || first {
			first = false // 首行视为表头
			continue
		}
		if len(rec) < 5 {
			continue
		}
		b, err1 := strconv.Atoi(rec[2])
		s, err2 := strconv.Atoi(rec[3])
		g, err3 := strconv.Atoi(rec[4])
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		draws = append(draws, Draw{Issue: rec[0], Date: rec[1], B: b, S: s, G: g})
	}
	return draws, nil
}

// AppendCSV 追加一期；若期号已存在返回 0（不追加），否则追加并返回 1。
// raw 列格式：b s g 后跟 12 个 0。
func AppendCSV(path string, d Draw) (int, error) {
	existing := map[string]bool{}
	if draws, err := LoadCSV(path); err == nil {
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
	// 新文件写表头（LoadCSV 会跳过第一行，无表头则首条数据被误当表头，幂等检测失效）
	if fi, err := f.Stat(); err == nil && fi.Size() == 0 {
		if _, err := f.WriteString("issue,date,hundreds,tens,ones,number,raw\n"); err != nil {
			return 0, err
		}
	}
	w := csv.NewWriter(f)
	raw := strconv.Itoa(d.B) + " " + strconv.Itoa(d.S) + " " + strconv.Itoa(d.G) + " 0 0 0 0 0 0 0 0 0 0 0 0"
	num := strconv.Itoa(d.B) + strconv.Itoa(d.S) + strconv.Itoa(d.G)
	if err := w.Write([]string{d.Issue, d.Date, strconv.Itoa(d.B), strconv.Itoa(d.S), strconv.Itoa(d.G), num, raw}); err != nil {
		return 0, err
	}
	w.Flush()
	return 1, w.Error()
}

// LastIssue 返回最新期号，空数据返回 ""。
func LastIssue(draws []Draw) string {
	if len(draws) == 0 {
		return ""
	}
	return draws[len(draws)-1].Issue
}
