// Package fetch 提供双色球开奖数据双源抓取（灰鸟 API 主 + 17500.cn 备份）、
// 期号合理性校验。复用 httpGet 与重试逻辑，仅解析格式不同。
package fetch

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fc3d-kill6/data"
)

// LatestSSQ 双色球最新一期
type LatestSSQ struct {
	Issue     string
	Date      string
	Reds      [6]int
	Blue      int
	NextIssue string // 数据源自带的下期期号（跨年安全），可为空
	Source    string
}

// FetchLatestSSQ 依次尝试双色球数据源，返回 (新数据, 源是否存活)。
// 期号必须 > 本地 CSV 最新期号，否则视为缓存/旧数据拒绝（防假数据）。
func FetchLatestSSQ(csvPath string) (*LatestSSQ, bool) {
	lastIssue := ""
	if draws, err := data.LoadSSQCSV(csvPath); err == nil {
		lastIssue = data.LastSSQIssue(draws)
	}

	alive := false
	sources := []struct {
		name string
		fn   func() (*LatestSSQ, error)
	}{
		{"灰鸟API(ssq)", fetchHuiniaoSSQ},
		{"17500.cn(ssq)", fetch17500SSQ},
	}
	for _, src := range sources {
		lt, err := src.fn()
		if err != nil || lt == nil {
			if err != nil {
				fmt.Printf("  ⚠️ %s: %v\n", src.name, err)
			}
			continue
		}
		alive = true
		if lastIssue != "" && lt.Issue <= lastIssue {
			fmt.Printf("  ⏭️ %s: 期号%s<=本地%s, 跳过(无新期, 源正常)\n", src.name, lt.Issue, lastIssue)
			continue
		}
		fmt.Printf("  ✅ %s: %s (%s) %s + 蓝%s\n", src.name, lt.Issue, lt.Date, ssqRedsFmt(lt.Reds), itoa2n(lt.Blue))
		return lt, true
	}
	return nil, alive
}

// fetchHuiniaoSSQ 灰鸟 API 双色球（数字为字符串格式 "01"）
func fetchHuiniaoSSQ() (*LatestSSQ, error) {
	const url = "http://api.huiniao.top/interface/home/lotteryHistory?type=ssq&page=1&limit=1"
	body, err := httpGet(url, "")
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Data struct {
				List []struct {
					Code     string `json:"code"`
					Day      string `json:"day"`
					One      string `json:"one"`
					Two      string `json:"two"`
					Three    string `json:"three"`
					Four     string `json:"four"`
					Five     string `json:"five"`
					Six      string `json:"six"`
					Seven    string `json:"seven"`
					NextCode string `json:"next_code"`
				} `json:"list"`
			} `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 1 || len(resp.Data.Data.List) == 0 {
		return nil, fmt.Errorf("灰鸟API(ssq) 返回异常 code=%d", resp.Code)
	}
	it := resp.Data.Data.List[0]
	reds, blue, err := parseSSQNums([]string{it.One, it.Two, it.Three, it.Four, it.Five, it.Six, it.Seven})
	if err != nil {
		return nil, fmt.Errorf("灰鸟API(ssq) %v", err)
	}
	return &LatestSSQ{Issue: it.Code, Date: it.Day, Reds: reds, Blue: blue, NextIssue: it.NextCode, Source: "灰鸟API"}, nil
}

// fetch17500SSQ 17500 双色球 TXT（每行: 期号 日期 红×6 蓝 + 附加），带 UA 轮换重试
func fetch17500SSQ() (*LatestSSQ, error) {
	const url = "https://www.17500.cn/getData/ssq.TXT"
	uas := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36",
		"Python-urllib/3.11",
	}
	var body []byte
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		body, lastErr = httpGet(url, uas[attempt%len(uas)])
		if lastErr == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return parseSSQLine(body)
}

// parseSSQLine 解析 17500 双色球 TXT：取最后一行前 9 字段（期号 日期 红×6 蓝）
func parseSSQLine(body []byte) (*LatestSSQ, error) {
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("17500(ssq) 无数据")
	}
	fields := strings.Fields(strings.TrimSpace(lines[len(lines)-1]))
	if len(fields) < 9 {
		return nil, fmt.Errorf("17500(ssq) 行字段不足: %d", len(fields))
	}
	if !regexp.MustCompile(`^20\d{5}$`).MatchString(fields[0]) {
		return nil, fmt.Errorf("17500(ssq) 行尾期号解析失败: %q", fields[0])
	}
	reds, blue, err := parseSSQNums(fields[2:9])
	if err != nil {
		return nil, fmt.Errorf("17500(ssq) %v", err)
	}
	return &LatestSSQ{Issue: fields[0], Date: fields[1], Reds: reds, Blue: blue, Source: "17500.cn"}, nil
}

// parseSSQNums 解析 7 个双色球数字（字符串 "01" 或 "1"）：前 6 红球 + 1 蓝球
func parseSSQNums(ss []string) ([6]int, int, error) {
	var reds [6]int
	for i := 0; i < 6; i++ {
		n, err := strconv.Atoi(strings.TrimSpace(ss[i]))
		if err != nil || n < 1 || n > 33 {
			return reds, 0, fmt.Errorf("红球解析失败: %q", ss[i])
		}
		reds[i] = n
	}
	b, err := strconv.Atoi(strings.TrimSpace(ss[6]))
	if err != nil || b < 1 || b > 16 {
		return reds, 0, fmt.Errorf("蓝球解析失败: %q", ss[6])
	}
	return reds, b, nil
}

// ssqRedsFmt 红球格式化为 "01,04,16,22,26,31"
func ssqRedsFmt(reds [6]int) string {
	parts := make([]string, 6)
	for i, n := range reds {
		parts[i] = itoa2n(n)
	}
	return strings.Join(parts, ",")
}

func itoa2n(n int) string {
	if n < 10 {
		return "0" + fmt.Sprint(n)
	}
	return fmt.Sprint(n)
}
