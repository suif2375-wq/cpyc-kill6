// Package monitor 提供 6 杀全中率趋势记录与表现预警。
// 独立于回测统计：趋势数据为页面趋势图与迭代决策提供依据。
package monitor

import (
	"encoding/json"
	"os"
	"time"
)

// 预警阈值
const (
	TrigBelowPct  = 70.0 // 滚动 100 期跌破阈值告警
	TrigMonthDrop = 8.0  // 单月下滑超过阈值告警
	TrigMonthDays = 30
	Kill6MaxKeep  = 400
)

// Entry kill6_history.json 的一条记录
type Entry struct {
	Issue string  `json:"issue"`
	Date  string  `json:"date"`
	Pct   float64 `json:"pct"`
}

// Record 把近 N 期 6 杀全中率追加到历史文件；同期号覆盖；只留最近 Kill6MaxKeep 条。
func Record(path string, pct float64, issue, date string) ([]Entry, error) {
	hist := []Entry{}
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &hist)
	}
	if len(hist) > 0 && hist[len(hist)-1].Issue == issue {
		hist[len(hist)-1].Pct = round1(pct)
		hist[len(hist)-1].Date = date
	} else {
		hist = append(hist, Entry{Issue: issue, Date: date, Pct: round1(pct)})
	}
	if len(hist) > Kill6MaxKeep {
		hist = hist[len(hist)-Kill6MaxKeep:]
	}
	raw, err := json.MarshalIndent(hist, "", " ")
	if err != nil {
		return hist, err
	}
	return hist, os.WriteFile(path, raw, 0o644)
}

// CheckAlert 检测两个表现预警条件，返回 (是否告警, 原因列表, 单月下滑pp)。
// 仅告警、不提供"升级建议"：算法穷举工具不在本仓库，悬空建议已移除。
func CheckAlert(curPct float64, hist []Entry) (bool, []string, float64) {
	reasons := []string{}
	if curPct < TrigBelowPct {
		reasons = append(reasons, formatPct(curPct)+" 跌破 "+formatPct(TrigBelowPct)+" 预警阈值")
	}
	drop := 0.0
	if len(hist) >= 2 {
		curD, err := time.Parse("2006-01-02", hist[len(hist)-1].Date)
		if err == nil {
			var ref *Entry
			for i := len(hist) - 2; i >= 0; i-- {
				hd, err2 := time.Parse("2006-01-02", hist[i].Date)
				if err2 != nil {
					continue
				}
				if curD.Sub(hd).Hours()/24 >= TrigMonthDays {
					ref = &hist[i]
					break
				}
			}
			if ref == nil {
				ref = &hist[0]
			}
			drop = ref.Pct - curPct
			if drop >= TrigMonthDrop {
				reasons = append(reasons, "单月下滑 "+formatPct(drop)+"pp (从"+ref.Date+"的"+formatPct(ref.Pct)+"%) 超过 "+formatPct(TrigMonthDrop)+"pp")
			}
		}
	}
	return len(reasons) > 0, reasons, round1(drop)
}

func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}

func formatPct(f float64) string {
	return itoa(int(f)) + "." + pad2(int(f*10)%10) + "%"
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
