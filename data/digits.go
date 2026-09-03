package data

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
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
	if len(d.Digits) == 0 {
		return 0, fmt.Errorf("digits must not be empty")
	}
	existing := map[string]bool{}
	if draws, err := LoadDigitCSV(path, len(d.Digits)); err == nil {
		for _, draw := range draws {
			existing[draw.Issue] = true
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
		header := make([]string, 0, len(d.Digits)+2)
		header = append(header, "issue", "date")
		for i := range d.Digits {
			header = append(header, fmt.Sprintf("d%d", i+1))
		}
		if _, err := f.WriteString(strings.Join(header, ",") + "\n"); err != nil {
			return 0, err
		}
	}
	w := csv.NewWriter(f)
	row := make([]string, 0, len(d.Digits)+2)
	row = append(row, d.Issue, d.Date)
	for _, n := range d.Digits {
		row = append(row, strconv.Itoa(n))
	}
	if err := w.Write(row); err != nil {
		return 0, err
	}
	w.Flush()
	return 1, w.Error()
}

// LastDigitIssue 返回最新期号。
func LastDigitIssue(draws []DigitDraw) string {
	if len(draws) == 0 {
		return ""
	}
	return draws[len(draws)-1].Issue
}
