package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

//go:embed template.html
var templateFS embed.FS

var benchLine = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op\s+([0-9.]+)\s+B/op\s+([0-9]+)\s+allocs/op$`)

type Sample struct {
	Case      string  // e.g., Basic, JSON, Middleware5, Params, HEAD_Explicit, OPTIONS_Explicit
	Framework string  // Slim, Gin, Echo, Fiber
	NsOp      float64 // nanoseconds per op
	BOp       float64 // bytes per op
	AllocsOp  int     // allocs per op
}

func parseFile(path string) ([]Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	var out []Sample
	for s.Scan() {
		m := benchLine.FindStringSubmatch(s.Text())
		if len(m) == 5 {
			name := m[1] // e.g., BenchmarkParams_Slim or BenchmarkHEAD_Explicit_Slim
			parts := strings.Split(name, "_")
			if len(parts) < 2 {
				continue
			}
			framework := parts[len(parts)-1]
			// base is the thing after "Benchmark" in the first part
			base := strings.TrimPrefix(parts[0], "Benchmark") // Params, HEAD, OPTIONS, ...
			middle := parts[1 : len(parts)-1]                   // e.g., [Explicit]
			var caseName string
			if len(middle) == 0 {
				caseName = base
			} else {
				caseName = base + "_" + strings.Join(middle, "_")
			}

			var nsop, bop float64
			var allocs int
			fmt.Sscanf(m[2], "%f", &nsop)
			fmt.Sscanf(m[3], "%f", &bop)
			fmt.Sscanf(m[4], "%d", &allocs)

			out = append(out, Sample{Case: caseName, Framework: framework, NsOp: nsop, BOp: bop, AllocsOp: allocs})
		}
	}
	return out, s.Err()
}

func groupByCase(samples []Sample) map[string][]Sample {
	m := make(map[string][]Sample)
	for _, sm := range samples {
		m[sm.Case] = append(m[sm.Case], sm)
	}
	return m
}

// aggregate averages repeated samples for the same (Case, Framework)
func aggregate(samples []Sample) []Sample {
	type acc struct{ ns, b float64; alloc float64; n int }
	accm := map[string]acc{}
	for _, s := range samples {
		k := s.Case + "\x00" + s.Framework
		a := accm[k]
		a.ns += s.NsOp
		a.b += s.BOp
		a.alloc += float64(s.AllocsOp)
		a.n++
		accm[k] = a
	}
	out := make([]Sample, 0, len(accm))
	for k, a := range accm {
		parts := strings.Split(k, "\x00")
		out = append(out, Sample{
			Case:      parts[0],
			Framework: parts[1],
			NsOp:      a.ns / float64(a.n),
			BOp:       a.b / float64(a.n),
			AllocsOp:  int(math.Round(a.alloc / float64(a.n))),
		})
	}
	return out
}

func uniqueFrameworks(samples []Sample) []string {
	set := map[string]struct{}{}
	for _, s := range samples {
		set[s.Framework] = struct{}{}
	}
	var list []string
	for k := range set {
		list = append(list, k)
	}
	sort.Strings(list)
	return list
}

func main() {
	outDir := filepath.Join("..", "..", "out")
	// Accept CWD anywhere under bench/, but write into bench/out
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		// try local out
		outDir = filepath.Join("out")
	}
	_ = os.MkdirAll(outDir, 0o755)

	files := []string{
		filepath.Join(outDir, "run_slim.txt"),
		filepath.Join(outDir, "run_gin.txt"),
		filepath.Join(outDir, "run_echo.txt"),
		filepath.Join(outDir, "run_fiber.txt"),
		filepath.Join(outDir, "run_chi.txt"),
	}

	var all []Sample
	for _, f := range files {
		sm, err := parseFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %v\n", err)
			continue
		}
		all = append(all, sm...)
	}
	if len(all) == 0 {
		fmt.Fprintln(os.Stderr, "no samples found; ensure run_*.txt exist under bench/out")
		os.Exit(1)
	}

	// Average repeated runs per case+framework (e.g., when using -count>1)
	all = aggregate(all)
	byCase := groupByCase(all)
	cases := make([]string, 0, len(byCase))
	for k := range byCase {
		cases = append(cases, k)
	}
	// Preferred case order if present
	preferredCases := []string{"Params", "HEAD_Explicit", "OPTIONS_Explicit"}
	seen := map[string]struct{}{}
	var ordered []string
	for _, p := range preferredCases {
		if _, ok := byCase[p]; ok {
			ordered = append(ordered, p)
			seen[p] = struct{}{}
		}
	}
	var rest []string
	for _, c := range cases {
		if _, ok := seen[c]; !ok {
			rest = append(rest, c)
		}
	}
	sort.Strings(rest)
	cases = append(ordered, rest...)

	// Stabilize framework order: prefer this order if present, then append unknowns sorted.
	preferred := []string{"Slim", "Gin", "Echo", "Fiber", "Chi"}
	set := map[string]struct{}{}
	for _, s := range all { set[s.Framework] = struct{}{} }
	var frameworks []string
	for _, p := range preferred {
		if _, ok := set[p]; ok { frameworks = append(frameworks, p) }
	}
	// collect unknowns
	var unknowns []string
	for k := range set {
		known := false
		for _, p := range preferred { if k == p { known = true; break } }
		if !known { unknowns = append(unknowns, k) }
	}
	sort.Strings(unknowns)
	frameworks = append(frameworks, unknowns...)

	htmlBytes, err := templateFS.ReadFile("template.html")
	if err != nil {
		panic(err)
	}
	html := string(htmlBytes)

	// Build JS data
	var sb strings.Builder
	sb.WriteString("const dataByCase = {\n")
	for _, c := range cases {
		sb.WriteString(fmt.Sprintf("  '%s': {\n", c))
		// Map framework -> sample
		m := map[string]Sample{}
		for _, sm := range byCase[c] {
			m[sm.Framework] = sm
		}
		// Ensure consistent order by frameworks slice
		nsOps := make([]string, 0, len(frameworks))
		bOps := make([]string, 0, len(frameworks))
		allocs := make([]string, 0, len(frameworks))
		for _, fw := range frameworks {
			if sm, ok := m[fw]; ok {
				nsOps = append(nsOps, fmt.Sprintf("%g", sm.NsOp))
				bOps = append(bOps, fmt.Sprintf("%g", sm.BOp))
				allocs = append(allocs, fmt.Sprintf("%d", sm.AllocsOp))
			} else {
				nsOps = append(nsOps, "null")
				bOps = append(bOps, "null")
				allocs = append(allocs, "null")
			}
		}
		// labels 使用 JSON 序列化，确保是合法的 JS 字符串数组
		if lb, err := json.Marshal(frameworks); err == nil {
			sb.WriteString(fmt.Sprintf("    labels: %s,\n", string(lb)))
		} else {
			// 退化：以逗号拼接（不太可能触发）
			sb.WriteString(fmt.Sprintf("    labels: [%s],\n", strings.Join(frameworks, ",")))
		}
		sb.WriteString(fmt.Sprintf("    nsop: [%s],\n", strings.Join(nsOps, ",")))
		sb.WriteString(fmt.Sprintf("    bop: [%s],\n", strings.Join(bOps, ",")))
		sb.WriteString(fmt.Sprintf("    allocs: [%s],\n", strings.Join(allocs, ",")))
		sb.WriteString("  },\n")
	}
	sb.WriteString("};\n")

	html = strings.ReplaceAll(html, "/*__DATA__*/", sb.String())

	outHTML := filepath.Join(outDir, "chart.html")
	if err := os.WriteFile(outHTML, []byte(html), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("Wrote:", outHTML)
}
