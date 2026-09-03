package data

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// DigitDraw 是排列3/排列5等数字型彩票的一期开奖。
// Digits 按开奖顺序保存，例如排列5为万、千、百、十、个位。
type DigitDraw struct {
	Issue  string
	Date   string
	Digits []int
}

// LoadDigitCSV 读取通用数字型彩票 CSV。文件格式为：
// issue,date,d1,d2,...,dN
func LoadDigitCSV(path string, positions int) ([]DigitDraw, error) {
	if positions <= 0 {
		return nil, fmt.Errorf("positions must be positive")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	draws := make([]DigitDraw, 0, 4096)
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
		if len(rec) < positions+2 {
			continue
		}
		digits := make([]int, positions)
		ok := true
		for i := 0; i < positions; i++ {
			n, e := strconv.Atoi(strings.TrimSpace(rec[i+2]))
			if e != nil || n < 0 || n > 9 {
				ok = false
				break
			}
			digits[i] = n
		}
		if ok {
			draws = append(draws, DigitDraw{Issue: rec[0], Date: rec[1], Digits: digits})
		}
	}
	return draws, nil
}

// AppendDigitCSV 追加一期通用数字型彩票数据，按期号幂等。
func AppendDigitCSV(path string, d DigitDraw) (int, error) {
	return AppendDigitCSVBatch(path, []DigitDraw{d})
}

// AppendDigitCSVBatch 批量幂等追加数字型彩票记录。
// 相比逐条调用 AppendDigitCSV，它只读取一次本地文件，适合首次同步数千期历史。
// 如果传入数据包含比本地最新期号更早的缺失记录，会合并后排序重写，保证回测时序正确。
func AppendDigitCSVBatch(path string, incoming []DigitDraw) (int, error) {
	if len(incoming) == 0 {
		return 0, nil
	}
	positions := len(incoming[0].Digits)
	if positions == 0 {
		return 0, fmt.Errorf("digits must not be empty")
	}
	existing, loadErr := LoadDigitCSV(path, positions)
	if loadErr != nil {
		existing = nil
	}
	seen := make(map[string]bool, len(existing)+len(incoming))
	for _, draw := range existing {
		seen[draw.Issue] = true
	}
	newDraws := make([]DigitDraw, 0, len(incoming))
	for _, draw := range incoming {
		if len(draw.Digits) != positions || seen[draw.Issue] {
			continue
		}
		seen[draw.Issue] = true
		newDraws = append(newDraws, draw)
	}
	if len(newDraws) == 0 {
		return 0, nil
	}
	sort.SliceStable(newDraws, func(i, j int) bool { return newDraws[i].Issue < newDraws[j].Issue })
	needsRewrite := len(existing) > 0 && newDraws[0].Issue <= existing[len(existing)-1].Issue
	if needsRewrite {
		all := append(append([]DigitDraw(nil), existing...), newDraws...)
		sort.SliceStable(all, func(i, j int) bool { return all[i].Issue < all[j].Issue })
		if err := writeDigitCSV(path, all, positions); err != nil {
			return 0, err
		}
		return len(newDraws), nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if fi, err := f.Stat(); err == nil && fi.Size() == 0 {
		header := make([]string, 0, positions+2)
		header = append(header, "issue", "date")
		for i := 0; i < positions; i++ {
			header = append(header, fmt.Sprintf("d%d", i+1))
		}
		if _, err := f.WriteString(strings.Join(header, ",") + "\n"); err != nil {
			return 0, err
		}
	}
	w := csv.NewWriter(f)
	for _, draw := range newDraws {
		row := make([]string, 0, positions+2)
		row = append(row, draw.Issue, draw.Date)
		for _, n := range draw.Digits {
			row = append(row, strconv.Itoa(n))
		}
		if err := w.Write(row); err != nil {
			return 0, err
		}
	}
	w.Flush()
	return len(newDraws), w.Error()
}

func writeDigitCSV(path string, draws []DigitDraw, positions int) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	header := make([]string, 0, positions+2)
	header = append(header, "issue", "date")
	for i := 0; i < positions; i++ {
		header = append(header, fmt.Sprintf("d%d", i+1))
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, draw := range draws {
		if len(draw.Digits) != positions {
			continue
		}
		row := make([]string, 0, positions+2)
		row = append(row, draw.Issue, draw.Date)
		for _, n := range draw.Digits {
			row = append(row, strconv.Itoa(n))
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// LastDigitIssue 返回最新期号。
func LastDigitIssue(draws []DigitDraw) string {
	if len(draws) == 0 {
		return ""
	}
	return draws[len(draws)-1].Issue
}
