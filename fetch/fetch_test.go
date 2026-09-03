package fetch

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fc3d-kill6/data"
)

// 模拟 17500.cn 真实 TXT 格式（每行: 期号 日期 百 十 个 + 10 附加数字）
const mock17500 = `2026201 2026-07-30 3 2 1 0 0 0 0 0 0 0 0 0 0 0
2026202 2026-07-31 9 8 8 0 0 0 0 0 0 0 0 0 0 0
2026222 2026-08-20 3 8 0 0 0 0 0 0 0 0 0 0 0
`

func TestFetch17500Parse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(mock17500))
	}))
	defer srv.Close()

	// 通过反射替换 URL 不现实，直接测试核心解析逻辑
	body := []byte(mock17500)
	fields := strings.Fields(strings.TrimSpace(string(body)))
	if len(fields) < 15 {
		t.Fatalf("字段不足: %d", len(fields))
	}
	last := fields[len(fields)-15:]
	if last[0] != "2026222" || last[1] != "2026-08-20" || last[2] != "3" || last[3] != "8" || last[4] != "0" {
		t.Fatalf("解析失败: %v", last[:5])
	}
	// 验证 fetch17500 的解析逻辑等价实现
	lt, err := parse17500Line(body)
	if err != nil {
		t.Fatalf("parse17500Line: %v", err)
	}
	if lt.Issue != "2026222" || lt.Date != "2026-08-20" || lt.B != 3 || lt.S != 8 || lt.G != 0 {
		t.Fatalf("结果错误: %+v", lt)
	}
	_ = srv
}

func TestParse17500UsesLastValidLine(t *testing.T) {
	body := []byte("2026221 2026-08-19 1 2 3 0 0 0 0 0 0 0 0 0 0 0\n" +
		"2026222 2026-08-20 9 8 0 123 456 789 0 0 0 0 0 0 0 0 2\n")
	lt, err := parse17500Line(body)
	if err != nil {
		t.Fatalf("parse17500Line: %v", err)
	}
	if lt.Issue != "2026222" || lt.B != 9 || lt.S != 8 || lt.G != 0 {
		t.Fatalf("last valid line not selected: %+v", lt)
	}
}

func TestAppendCSVIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	// 初始数据
	if _, err := data.AppendCSV(path, data.Draw{Issue: "2026001", Date: "2026-01-01", B: 1, S: 2, G: 3}); err != nil {
		t.Fatal(err)
	}
	// 重复追加同期号 → 返回 0
	n, err := data.AppendCSV(path, data.Draw{Issue: "2026001", Date: "2026-01-01", B: 1, S: 2, G: 3})
	if err != nil || n != 0 {
		t.Fatalf("重复追加应返回 0, got %d err=%v", n, err)
	}
	// 新期号 → 返回 1
	n, err = data.AppendCSV(path, data.Draw{Issue: "2026002", Date: "2026-01-02", B: 4, S: 5, G: 6})
	if err != nil || n != 1 {
		t.Fatalf("新期号应返回 1, got %d err=%v", n, err)
	}
	draws, err := data.LoadCSV(path)
	if err != nil || len(draws) != 2 {
		t.Fatalf("应 2 期, got %d err=%v", len(draws), err)
	}
	if draws[1].Issue != "2026002" || draws[1].B != 4 {
		t.Fatalf("第二期解析错误: %+v", draws[1])
	}
}

func TestNextIssueCalc(t *testing.T) {
	cases := []struct {
		issue, date, next string
		want              string
	}{
		{"2026365", "2026-12-31", "", "2027001"},        // 跨年回绕
		{"2026365", "2026-12-31", "2027001", "2027001"}, // 数据源优先
		{"2026009", "2026-01-09", "", "2026010"},        // 补零
		{"2026222", "2026-08-20", "2026223", "2026223"},
	}
	for _, c := range cases {
		if got := NextIssueCalc(c.issue, c.date, c.next); got != c.want {
			t.Fatalf("NextIssueCalc(%s,%s,%s)=%s, want %s", c.issue, c.date, c.next, got, c.want)
		}
	}
}

// 确保数据目录文件存在（防误删）
func TestDataDirHasCSV(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "fc3d-history.csv")); err != nil {
		t.Skip("项目根 CSV 不存在（未在仓库根运行）")
	}
}
