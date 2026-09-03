package engine

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// golden 数据结构（tools/gengolden 输出）
type goldenFile struct {
	Source     string        `json:"source"`
	Total      int           `json:"total"`
	Rows       []goldenRow   `json:"rows"`
	Exhaustive []goldenExact `json:"exhaustive"`
}

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

type draw struct {
	Issue   string
	B, S, G int
}

func loadGolden(t *testing.T) goldenFile {
	t.Helper()
	root, _ := os.Getwd()
	// 兼容从 engine/ 或项目根运行
	path := filepath.Join(root, "golden.json")
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join(filepath.Dir(root), "golden.json")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 golden.json 失败: %v", err)
	}
	var g goldenFile
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("解析 golden.json 失败: %v", err)
	}
	return g
}

func loadCSV(t *testing.T) []draw {
	t.Helper()
	root, _ := os.Getwd()
	path := filepath.Join(root, "fc3d-history.csv")
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join(filepath.Dir(root), "fc3d-history.csv")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("读取 CSV 失败: %v", err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	recs, err := r.ReadAll()
	if err != nil {
		t.Fatalf("解析 CSV 失败: %v", err)
	}
	draws := make([]draw, 0, len(recs))
	for i, rec := range recs {
		if i == 0 || len(rec) < 5 {
			continue
		}
		draws = append(draws, draw{rec[0], atoi(rec[2]), atoi(rec[3]), atoi(rec[4])})
	}
	return draws
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// TestGoldenRows 逐期一致性：Go 状态机 vs Python golden（8729 期）
func TestGoldenRows(t *testing.T) {
	g := loadGolden(t)
	draws := loadCSV(t)
	if g.Total != len(draws)-1 {
		t.Fatalf("golden 期数 %d != CSV 期数-1 %d", g.Total, len(draws)-1)
	}

	st := NewState()
	for i := 1; i < len(draws); i++ {
		p := draws[i-1]
		hk, tk, ok, hk2, tk2, ok2 := st.Next(p.B, p.S, p.G, draws[i].G)
		want := g.Rows[i-1]
		if hk != want.HK || tk != want.TK || ok != want.OK ||
			hk2 != want.HK2 || tk2 != want.TK2 || ok2 != want.OK2 {
			t.Fatalf("第 %d 期 (%s) 不一致:\n  got  百=%d 十=%d 个=%d | k2 %d,%d,%d\n  want 百=%d 十=%d 个=%d | k2 %d,%d,%d",
				i, want.Issue, hk, tk, ok, hk2, tk2, ok2,
				want.HK, want.TK, want.OK, want.HK2, want.TK2, want.OK2)
		}
	}
	t.Logf("✅ 逐期一致: %d 期全部通过 (Go 引擎 == 引擎基准数据)", g.Total)
}

// TestGoldenExhaustive 纯函数全量穷举：1000 组 (0-9)³ 输入，6 个纯函数完全一致
func TestGoldenExhaustive(t *testing.T) {
	g := loadGolden(t)
	for i, e := range g.Exhaustive {
		h, tt, o := KillH(e.B, e.S, e.G), KillT(e.B, e.S, e.G), KillO(e.B, e.S, e.G, nil, 0)
		h2, t2, o2 := KillH2(e.B, e.S, e.G), KillT2(e.B, e.S, e.G), KillO2(e.B, e.S, e.G)
		if h != e.H || tt != e.T || o != e.O || h2 != e.H2 || t2 != e.T2 || o2 != e.O2 {
			t.Fatalf("输入 (%d,%d,%d) 第 %d 组不一致: got h=%d t=%d o=%d k2=%d,%d,%d; want h=%d t=%d o=%d k2=%d,%d,%d",
				e.B, e.S, e.G, i, h, tt, o, h2, t2, o2, e.H, e.T, e.O, e.H2, e.T2, e.O2)
		}
	}
	t.Logf("✅ 纯函数穷举一致: %d 组全部通过", len(g.Exhaustive))
}

// TestMod10 验证负数取模修复的必要性
func TestMod10(t *testing.T) {
	cases := []struct{ in, want int }{
		{-7, 3}, {-10, 0}, {-1, 9}, {0, 0}, {13, 3}, {20, 0},
	}
	for _, c := range cases {
		if got := mod10(c.in); got != c.want {
			t.Fatalf("mod10(%d)=%d, want %d", c.in, got, c.want)
		}
	}
}
