package fetch

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"fc3d-kill6/data"
)

// DigitSource 描述一个排列3/排列5数据源。
type DigitSource struct {
	Name      string
	URL       string
	Positions int
}

var digitSources = map[string]DigitSource{
	"p3": {Name: "17500.cn 排列3", URL: "https://www.17500.cn/getData/p3.TXT", Positions: 3},
	"p5": {Name: "17500.cn 排列5", URL: "https://www.17500.cn/getData/p5.TXT", Positions: 5},
}

var digitIssueRE = regexp.MustCompile(`^20\d{5}$`)

// SyncDigits 下载完整历史并幂等追加到本地 CSV。
// 完整历史而不是只抓最新一期，便于首次部署就能做足滚动回测。
func SyncDigits(kind, csvPath string) (added int, source string, err error) {
	src, ok := digitSources[strings.ToLower(strings.TrimSpace(kind))]
	if !ok {
		return 0, "", fmt.Errorf("unknown digit kind %q", kind)
	}
	var body []byte
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		body, lastErr = httpGet(src.URL, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126 Safari/537.36")
		if lastErr == nil {
			break
		}
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	if lastErr != nil {
		return 0, src.Name, lastErr
	}
	draws, err := parseDigitTXT(body, src.Positions)
	if err != nil {
		return 0, src.Name, err
	}
	for _, draw := range draws {
		n, e := data.AppendDigitCSV(csvPath, draw)
		if e != nil {
			return added, src.Name, e
		}
		added += n
	}
	return added, src.Name, nil
}

// ParseDigitTXT 解析 17500.cn 的排列3/排列5 TXT。
// 每行前两列为期号、日期，随后为 N 个开奖号码，后面附加字段全部忽略。
func ParseDigitTXT(body []byte, positions int) ([]data.DigitDraw, error) {
	return parseDigitTXT(body, positions)
}

func parseDigitTXT(body []byte, positions int) ([]data.DigitDraw, error) {
	if positions <= 0 {
		return nil, fmt.Errorf("positions must be positive")
	}
	lines := strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")
	out := make([]data.DigitDraw, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < positions+2 || !digitIssueRE.MatchString(fields[0]) {
			continue
		}
		digits := make([]int, positions)
		ok := true
		for i := 0; i < positions; i++ {
			if len(fields[i+2]) != 1 || fields[i+2][0] < '0' || fields[i+2][0] > '9' {
				ok = false
				break
			}
			digits[i] = int(fields[i+2][0] - '0')
		}
		if ok {
			out = append(out, data.DigitDraw{Issue: fields[0], Date: fields[1], Digits: digits})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid %d-position draws found", positions)
	}
	return out, nil
}
