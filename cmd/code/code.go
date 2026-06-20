package code

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	codeLang    string
	codeDryRun  bool
	codeList    bool
	codeResults bool
	codeTimeout int
	codeCwd     string
	codeDir     string
	codeNoColor bool
)

// runner describes how to execute a fenced block language.
type runner struct {
	ext   string
	check func() string // returns interpreter path or "" if unavailable
	build func(tmp string) string
}

var fenceRe = regexp.MustCompile("^\\s{0,3}(`{3,}|~{3,})(.*)$")
var closeRe = regexp.MustCompile("^\\s{0,3}(`{3,}|~{3,})\\s*$")

var nonRunnable = map[string]bool{
	"": true, "json": true, "yaml": true, "yml": true, "toml": true, "ini": true,
	"html": true, "css": true, "scss": true, "less": true, "xml": true, "svg": true,
	"markdown": true, "md": true, "mdx": true, "rst": true, "asciidoc": true,
	"text": true, "txt": true, "plaintext": true, "log": true, "diff": true,
	"shell": true, "console": true, "dockerfile": true, "makefile": true,
	"gitignore": true, "gitconfig": true, "properties": true, "csv": true, "tsv": true,
	"graphql": true, "proto": true, "terraform": true, "hcl": true,
	"vim": true, "vimdoc": true,
}

func which(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func runners() map[string]runner {
	py := which("python3")
	if py == "" {
		py = which("python")
	}
	lua := which("lua")
	if lua == "" {
		lua = which("lua5.4")
	}
	if lua == "" {
		lua = which("lua5.3")
	}
	ts := which("ts-node")
	var tsBuild func(string) string
	if ts != "" {
		tsBuild = func(tmp string) string { return "ts-node " + tmp }
	} else if which("npx") != "" {
		tsBuild = func(tmp string) string { return "npx -y tsx " + tmp }
	} else {
		ts = which("tsx")
		tsBuild = func(tmp string) string { return "tsx " + tmp }
	}
	pwsh := which("pwsh")
	if pwsh == "" {
		pwsh = which("powershell")
	}
	scheme := which("guile")
	var schemeBuild func(string) string
	if scheme != "" {
		schemeBuild = func(tmp string) string { return "guile -s " + tmp }
	} else {
		scheme = which("gosh")
		schemeBuild = func(tmp string) string { return "gosh " + tmp }
	}
	hs := which("runghc")
	var hsBuild func(string) string
	if hs != "" {
		hsBuild = func(tmp string) string { return "runghc " + tmp }
	} else {
		hs = which("runhaskell")
		hsBuild = func(tmp string) string { return "runhaskell " + tmp }
	}

	return map[string]runner{
		"lua":        {ext: "lua", check: func() string { return lua }, build: func(tmp string) string { return lua + " " + tmp }},
		"python":     {ext: "py", check: func() string { return py }, build: func(tmp string) string { return py + " " + tmp }},
		"py":         {ext: "py", check: func() string { return py }, build: func(tmp string) string { return py + " " + tmp }},
		"bash":       {ext: "sh", check: func() string { return which("bash") }, build: func(tmp string) string { return "bash " + tmp }},
		"sh":         {ext: "sh", check: func() string { return which("sh") }, build: func(tmp string) string { return "sh " + tmp }},
		"zsh":        {ext: "zsh", check: func() string { return which("zsh") }, build: func(tmp string) string { return "zsh " + tmp }},
		"fish":       {ext: "fish", check: func() string { return which("fish") }, build: func(tmp string) string { return "fish " + tmp }},
		"ruby":       {ext: "rb", check: func() string { return which("ruby") }, build: func(tmp string) string { return "ruby " + tmp }},
		"rb":         {ext: "rb", check: func() string { return which("ruby") }, build: func(tmp string) string { return "ruby " + tmp }},
		"javascript": {ext: "js", check: func() string { return which("node") }, build: func(tmp string) string { return "node " + tmp }},
		"js":         {ext: "js", check: func() string { return which("node") }, build: func(tmp string) string { return "node " + tmp }},
		"typescript": {ext: "ts", check: func() string { return ts }, build: tsBuild},
		"ts":         {ext: "ts", check: func() string { return ts }, build: tsBuild},
		"go":         {ext: "go", check: func() string { return which("go") }, build: func(tmp string) string { return "go run " + tmp }},
		"rust": {
			ext:   "rs",
			check: func() string { return which("rustc") },
			build: func(tmp string) string {
				out := strings.TrimSuffix(tmp, ".rs")
				return "rustc -O -o " + out + " " + tmp + " && " + out + " ; rm -f " + out
			},
		},
		"rs": {
			ext:   "rs",
			check: func() string { return which("rustc") },
			build: func(tmp string) string {
				out := strings.TrimSuffix(tmp, ".rs")
				return "rustc -O -o " + out + " " + tmp + " && " + out + " ; rm -f " + out
			},
		},
		"perl":       {ext: "pl", check: func() string { return which("perl") }, build: func(tmp string) string { return "perl " + tmp }},
		"php":        {ext: "php", check: func() string { return which("php") }, build: func(tmp string) string { return "php " + tmp }},
		"r":          {ext: "R", check: func() string { return which("Rscript") }, build: func(tmp string) string { return "Rscript " + tmp }},
		"rscript":    {ext: "R", check: func() string { return which("Rscript") }, build: func(tmp string) string { return "Rscript " + tmp }},
		"julia":      {ext: "jl", check: func() string { return which("julia") }, build: func(tmp string) string { return "julia " + tmp }},
		"awk":        {ext: "awk", check: func() string { return which("awk") }, build: func(tmp string) string { return "awk -f " + tmp }},
		"scheme":     {ext: "scm", check: func() string { return scheme }, build: schemeBuild},
		"clojure":    {ext: "clj", check: func() string { return which("clojure") }, build: func(tmp string) string { return "clojure " + tmp }},
		"haskell":    {ext: "hs", check: func() string { return hs }, build: hsBuild},
		"hs":         {ext: "hs", check: func() string { return hs }, build: hsBuild},
		"elixir":     {ext: "exs", check: func() string { return which("elixir") }, build: func(tmp string) string { return "elixir " + tmp }},
		"erlang":     {ext: "erl", check: func() string { return which("escript") }, build: func(tmp string) string { return "escript " + tmp }},
		"powershell": {ext: "ps1", check: func() string { return pwsh }, build: func(tmp string) string { return pwsh + " -File " + tmp }},
		"pwsh":       {ext: "ps1", check: func() string { return which("pwsh") }, build: func(tmp string) string { return "pwsh -File " + tmp }},
	}
}

// headers holds parsed org-babel-style `:key value` header args.
type headers struct {
	vars   map[string]string // raw NAME=VALUE strings (coerced at injection)
	tangle string            // "" = not tangled; "yes" = derive name; else path
	mkdirp bool
	noweb  bool
	name   string
}

// block is a parsed fenced code block.
type block struct {
	lang      string
	info      string
	body      string
	startLine int
	endLine   int
	index     int
	name      string
	hdr       headers
}

// tokenizeInfo splits an info string on whitespace, keeping quoted tokens.
func tokenizeInfo(info string) []string {
	var toks []string
	s := strings.TrimRight(info, " \t\r")
	i := 0
	for i < len(s) {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] == '"' || s[i] == '\'' {
			q := s[i]
			j := i + 1
			for j < len(s) && s[j] != q {
				j++
			}
			toks = append(toks, s[i+1:j])
			i = j + 1
		} else {
			j := i
			for j < len(s) && s[j] != ' ' && s[j] != '\t' {
				j++
			}
			toks = append(toks, s[i:j])
			i = j
		}
	}
	return toks
}

// parseHeaders parses org-babel-style `:key value` header args from a fence
// info string (the first token, the language, is skipped).
func parseHeaders(info string) headers {
	h := headers{vars: map[string]string{}}
	tokens := tokenizeInfo(info)
	for k := 1; k < len(tokens); k++ {
		t := tokens[k]
		if len(t) == 0 || t[0] != ':' {
			continue
		}
		key := strings.ToLower(t[1:])
		val := ""
		hasVal := false
		if k+1 < len(tokens) && tokens[k+1] != "" && tokens[k+1][0] != ':' {
			val = tokens[k+1]
			hasVal = true
			k++
		}
		switch key {
		case "tangle":
			if hasVal {
				h.tangle = val
			} else {
				h.tangle = "yes"
			}
		case "mkdirp":
			h.mkdirp = val == "yes" || val == "true" || !hasVal
		case "noweb":
			h.noweb = val == "yes" || val == "true" || val == "tangle"
		case "name":
			h.name = val
		case "var":
			if hasVal {
				if eq := strings.Index(val, "="); eq > 0 {
					h.vars[val[:eq]] = val[eq+1:]
				}
			}
		}
	}
	return h
}

// blockName returns the block's name (header `:name` wins, else a preceding
// `#+name:` / `<!-- down:name: -->` line).
func blockName(b block, lines []string) string {
	if b.hdr.name != "" {
		return b.hdr.name
	}
	if b.startLine >= 2 {
		prev := strings.TrimSpace(lines[b.startLine-2])
		if n := strings.TrimPrefix(prev, "#+name:"); n != prev {
			return strings.TrimSpace(n)
		}
		if n := strings.TrimPrefix(prev, "<!-- down:name:"); n != prev {
			return strings.TrimSuffix(strings.TrimSpace(n), "-->")
		}
	}
	return ""
}

func parseBlocks(text string) []block {
	var blocks []block
	lines := strings.Split(text, "\n")
	i := 0
	idx := 0
	for i < len(lines) {
		m := fenceRe.FindStringSubmatch(lines[i])
		if m == nil {
			i++
			continue
		}
		fence := m[1]
		fenceLen := len(fence)
		info := strings.TrimRight(m[2], " \t\r")
		fields := strings.Fields(info)
		langStr := ""
		if len(fields) > 0 {
			langStr = strings.ToLower(fields[0])
		}
		start := i
		var body []string
		j := i + 1
		closed := false
		for j < len(lines) {
			cm := closeRe.FindStringSubmatch(lines[j])
			if cm != nil && len(cm[1]) >= fenceLen {
				closed = true
				break
			}
			body = append(body, lines[j])
			j++
		}
		idx++
		b := block{
			lang:      langStr,
			info:      info,
			body:      strings.Join(body, "\n"),
			startLine: start + 1,
			endLine:   ternary(closed, j+1, len(lines)+1),
			index:     idx,
		}
		b.hdr = parseHeaders(info)
		b.name = blockName(b, lines)
		blocks = append(blocks, b)
		if closed {
			i = j + 1
		} else {
			i = len(lines)
		}
	}
	return blocks
}

func ternary(cond bool, a, b int) int {
	if cond {
		return a
	}
	return b
}

func runnerFor(lang string, rs map[string]runner) (runner, bool) {
	lang = strings.ToLower(lang)
	if nonRunnable[lang] {
		return runner{}, false
	}
	r, ok := rs[lang]
	return r, ok
}

func isRunnable(lang string, rs map[string]runner) bool {
	r, ok := runnerFor(lang, rs)
	if !ok {
		return false
	}
	if r.check != nil {
		return r.check() != ""
	}
	return true
}

// indexByName builds a name -> block index for noweb references.
func indexByName(blocks []block) map[string]block {
	m := map[string]block{}
	for _, b := range blocks {
		if b.name != "" {
			m[b.name] = b
		}
	}
	return m
}

var nowebRe = regexp.MustCompile(`<<\s*([\w_-]+)\s*>>`)

// expandNoweb replaces `<<name>>` references with the referenced block body.
func expandNoweb(body string, byName map[string]block) string {
	return nowebRe.ReplaceAllStringFunc(body, func(m string) string {
		sub := nowebRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		if ref, ok := byName[sub[1]]; ok {
			return ref.body
		}
		return m
	})
}

// coerceVar parses a `:var` value string into a typed Go value.
func coerceVar(v string) interface{} {
	if v == "true" {
		return true
	}
	if v == "false" {
		return false
	}
	if n, err := strconv.ParseFloat(v, 64); err == nil {
		return n
	}
	return v
}

// varFormatter returns a language-specific assignment line for a variable.
func varFormatter(lang string) func(name string, v interface{}) string {
	quote := func(s string) string {
		return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
	}
	lit := func(v interface{}) string {
		switch x := v.(type) {
		case string:
			return quote(x)
		case bool:
			if lang == "python" || lang == "py" {
				if x {
					return "True"
				}
				return "False"
			}
			return strconv.FormatBool(x)
		default:
			return fmt.Sprintf("%v", x)
		}
	}
	formatters := map[string]func(string, interface{}) string{
		"python": func(n string, v interface{}) string { return n + " = " + lit(v) },
		"py":     func(n string, v interface{}) string { return n + " = " + lit(v) },
		"lua":    func(n string, v interface{}) string { return n + " = " + lit(v) },
		"bash": func(n string, v interface{}) string {
			if s, ok := v.(string); ok {
				return n + "='" + strings.ReplaceAll(s, "'", "'\\''") + "'"
			}
			return n + "=" + fmt.Sprintf("%v", v)
		},
		"sh":         func(n string, v interface{}) string { return varFormatter("bash")(n, v) },
		"zsh":        func(n string, v interface{}) string { return varFormatter("bash")(n, v) },
		"ruby":       func(n string, v interface{}) string { return n + " = " + lit(v) },
		"rb":         func(n string, v interface{}) string { return n + " = " + lit(v) },
		"javascript": func(n string, v interface{}) string { return "let " + n + " = " + lit(v) },
		"js":         func(n string, v interface{}) string { return "let " + n + " = " + lit(v) },
		"typescript": func(n string, v interface{}) string { return "let " + n + " = " + lit(v) },
		"ts":         func(n string, v interface{}) string { return "let " + n + " = " + lit(v) },
		"go": func(n string, v interface{}) string {
			if s, ok := v.(string); ok {
				return n + " := " + quote(s)
			}
			return n + " := " + fmt.Sprintf("%v", v)
		},
		"perl": func(n string, v interface{}) string {
			return "my $" + n + " = " + lit(v) + ";"
		},
	}
	if f, ok := formatters[lang]; ok {
		return f
	}
	return func(n string, v interface{}) string { return "# down:var " + n + " = " + fmt.Sprintf("%v", v) }
}

// injectVars prepends `:var` assignments (language-specific) to a body.
func injectVars(body, lang string, vars map[string]string) string {
	if len(vars) == 0 {
		return body
	}
	fmtFn := varFormatter(lang)
	var pre []string
	for n, raw := range vars {
		pre = append(pre, fmtFn(n, coerceVar(raw)))
	}
	return strings.Join(pre, "\n") + "\n" + body
}

// prepareBlock returns the executable body (noweb expansion + var injection).
func prepareBlock(b block, blocks []block) string {
	body := b.body
	byName := indexByName(blocks)
	if b.hdr.noweb || nowebRe.MatchString(body) {
		body = expandNoweb(body, byName)
	}
	if len(b.hdr.vars) > 0 {
		body = injectVars(body, b.lang, b.hdr.vars)
	}
	return body
}

// tangleTarget resolves the tangle output path for a block (nil if not tangled).
func tangleTarget(b block, sourcePath string) string {
	t := b.hdr.tangle
	if t == "" {
		return ""
	}
	if t == "yes" || t == "true" {
		base := "tangled"
		if sourcePath != "" {
			if i := strings.LastIndex(sourcePath, "/"); i >= 0 {
				sourcePath = sourcePath[i+1:]
			}
			if j := strings.LastIndex(sourcePath, "."); j > 0 {
				sourcePath = sourcePath[:j]
			}
			base = sourcePath
		}
		r, _ := runnerFor(b.lang, runners())
		ext := r.ext
		if ext == "" {
			ext = b.lang
			if ext == "" {
				ext = "txt"
			}
		}
		return base + "." + ext
	}
	return t
}

// tangleText writes every `:tangle` block in `text` to a file. Returns written
// paths. noweb expansion is applied to tangled bodies.
func tangleText(text, sourcePath string, dir string) []string {
	blocks := parseBlocks(text)
	var written []string
	for _, b := range blocks {
		target := tangleTarget(b, sourcePath)
		if target == "" {
			continue
		}
		if dir != "" && !strings.HasPrefix(target, "/") {
			target = filepath.Join(dir, target)
		}
		if d := filepath.Dir(target); d != "." && d != "/" {
			if b.hdr.mkdirp || dir != "" {
				_ = os.MkdirAll(d, 0755)
			}
		}
		body := prepareBlock(b, blocks)
		if err := os.WriteFile(target, []byte(body), 0644); err == nil {
			written = append(written, target)
		}
	}
	return written
}

// isResultBlock reports whether a block is a down results block.
func isResultBlock(b block) bool {
	return strings.Contains(b.info, ":down_result")
}

// resultName extracts the name from a result block's info string.
func resultName(b block) string {
	i := strings.Index(b.info, ":down_result")
	if i < 0 {
		return ""
	}
	rest := strings.TrimSpace(b.info[i+len(":down_result"):])
	if sp := strings.IndexAny(rest, " \t"); sp >= 0 {
		rest = rest[:sp]
	}
	return rest
}

// resultLines builds the fenced result block lines for a run result.
func resultLines(b block, output string) []string {
	name := b.name
	if name == "" {
		name = "#" + strconv.Itoa(b.index)
	}
	lines := []string{"```text :down_result " + name}
	if strings.TrimSpace(output) != "" {
		lines = append(lines, strings.Split(strings.TrimRight(output, "\n"), "\n")...)
	}
	lines = append(lines, "```")
	return lines
}

// applyResults inserts/replaces `:down_result` blocks beneath each source
// block. resultsMap is keyed by block index.
func applyResults(text string, resultsMap map[int]string) string {
	blocks := parseBlocks(text)
	lines := strings.Split(text, "\n")
	for i := len(blocks) - 1; i >= 0; i-- {
		b := blocks[i]
		out, ok := resultsMap[b.index]
		if !ok {
			continue
		}
		ridx := 0
		if i+1 < len(blocks) && isResultBlock(blocks[i+1]) {
			ridx = i + 1
		} else {
			name := b.name
			for k := i + 1; k < len(blocks); k++ {
				if isResultBlock(blocks[k]) && (name == "" || resultName(blocks[k]) == name) {
					ridx = k
					break
				}
				if !isResultBlock(blocks[k]) {
					break
				}
			}
		}
		newLines := resultLines(b, out)
		if ridx > 0 {
			rb := blocks[ridx]
			start := rb.startLine - 1 // 0-indexed
			end := rb.endLine         // exclusive
			lines = append(lines[:start], append(newLines, lines[end:]...)...)
		} else {
			pos := b.endLine // 0-indexed insert after closing fence
			ins := append([]string{""}, newLines...)
			lines = append(lines[:pos], append(ins, lines[pos:]...)...)
		}
	}
	return strings.Join(lines, "\n")
}

func writeTemp(b block, r runner, body string) (string, error) {
	ext := r.ext
	if ext == "" {
		ext = b.lang
		if ext == "" {
			ext = "txt"
		}
	}
	dir := os.TempDir()
	name := fmt.Sprintf("down-code-%d.%s", os.Getpid(), ext)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func runBlock(b block, rs map[string]runner, blocks []block) (output string, exitCode int, cmdStr string, skipped bool) {
	r, ok := runnerFor(b.lang, rs)
	if !ok {
		skipped = true
		return
	}
	if r.check != nil && r.check() == "" {
		skipped = true
		return
	}
	body := prepareBlock(b, blocks)
	tmp, err := writeTemp(b, r, body)
	if err != nil {
		skipped = true
		return
	}
	defer os.Remove(tmp)
	cmdStr = r.build(tmp)
	full := cmdStr + " 2>&1"
	if codeCwd != "" {
		full = "cd " + codeCwd + " && " + full
	}
	if codeTimeout > 0 && which("timeout") != "" {
		full = fmt.Sprintf("timeout %d %s", codeTimeout, full)
	}
	if codeDryRun {
		return "", 0, cmdStr, false
	}
	c := exec.Command("sh", "-c", full)
	var sb strings.Builder
	pr, pw, _ := os.Pipe()
	c.Stdout = pw
	c.Stderr = pw
	if err := c.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "  error: %v\n", err)
		return "", 1, cmdStr, false
	}
	pw.Close()
	scan := bufio.NewScanner(pr)
	scan.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scan.Scan() {
		sb.WriteString(scan.Text())
		sb.WriteByte('\n')
	}
	err = c.Wait()
	if ee, ok := err.(*exec.ExitError); ok {
		exitCode = ee.ExitCode()
	} else if err != nil {
		exitCode = 1
	} else {
		exitCode = 0
	}
	return sb.String(), exitCode, cmdStr, false
}

// Code is the cobra command registered by the down CLI.
var Code = cobra.Command{
	Use:   "code [options] <file>",
	Short: "Run runnable fenced code blocks in a markdown file",
	Long: `Discover and execute every runnable fenced code block in a markdown file,
mirroring org-mode's "execute code block" workflow.

  :tangle FILE|yes    Write this block to FILE (yes => <source>.<ext>)
  :mkdirp yes          Create parent dirs when tangling
  :var NAME=VALUE      Inject a variable (string/number/bool) before running
  :noweb yes|tangle    Expand <<name>> references from named blocks
  :name NAME           Name this block for noweb references

A block is only run when its info-string language has a matching interpreter
on PATH. Markup/data languages (json, yaml, html, ...) are never executed.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: down code [options] <file>")
			os.Exit(1)
		}
		path := args[0]
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "down code: cannot read %s: %v\n", path, err)
			os.Exit(1)
		}
		blocks := parseBlocks(string(data))
		rs := runners()

		if codeList {
			fmt.Printf("%s: %d fenced block(s)\n", path, len(blocks))
			for _, b := range blocks {
				mark := " "
				if isRunnable(b.lang, rs) {
					mark = "▶"
				}
				tail := ""
				if b.name != "" {
					tail = " name=" + b.name
				}
				fmt.Printf("  %s [%s] #%d line %d%s\n", mark, b.lang, b.index, b.startLine, tail)
			}
			return
		}

		resultsMap := map[int]string{}
		ran, failed := 0, 0
		for _, b := range blocks {
			if codeLang != "" && b.lang != strings.ToLower(codeLang) {
				continue
			}
			if !isRunnable(b.lang, rs) {
				continue
			}
			out, ec, cmdStr, skipped := runBlock(b, rs, blocks)
			ran++
			fmt.Printf("▶ [%s] block #%d (line %d)\n", b.lang, b.index, b.startLine)
			if skipped {
				fmt.Println("  skipped (no runner or interpreter unavailable)")
				continue
			}
			if codeDryRun {
				fmt.Println("  $ " + cmdStr)
				continue
			}
			if strings.TrimSpace(out) != "" {
				for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
					fmt.Println("  " + ln)
				}
			}
			resultsMap[b.index] = out
			if ec == 0 {
				fmt.Println("  ✓ exit 0")
			} else {
				fmt.Printf("  ✗ exit %d\n", ec)
				failed++
			}
		}
		if codeResults && ran > 0 {
			newText := applyResults(string(data), resultsMap)
			if err := os.WriteFile(path, []byte(newText), 0644); err == nil {
				fmt.Printf("wrote results back to %s\n", path)
			}
		}
		if ran == 0 {
			fmt.Printf("No runnable code blocks found in %s\n", path)
			return
		}
		fmt.Printf("\n%d block(s) run, %d failed\n", ran, failed)
		if failed != 0 {
			os.Exit(1)
		}
	},
}

// CodeTangle writes ":tangle" blocks to files.
var CodeTangle = cobra.Command{
	Use:   "tangle [options] <file>",
	Short: "Write :tangle code blocks to files (org-babel tangle)",
	Long: `Extract every fenced code block carrying a :tangle header arg into a
file, mirroring org-babel's tangle. ":tangle yes" derives a name from the
source basename plus the language extension; ":mkdirp yes" creates parent
directories.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: down code tangle [options] <file>")
			os.Exit(1)
		}
		path := args[0]
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "down code tangle: cannot read %s: %v\n", path, err)
			os.Exit(1)
		}
		written := tangleText(string(data), path, codeDir)
		if len(written) == 0 {
			fmt.Printf("No `:tangle` blocks found in %s\n", path)
			return
		}
		for _, p := range written {
			fmt.Println("tangled -> " + p)
		}
		fmt.Printf("\n%d file(s) tangled\n", len(written))
	},
}

func init() {
	Code.Flags().StringVarP(&codeLang, "lang", "l", "", "Only run blocks of this language")
	Code.Flags().BoolVar(&codeDryRun, "dry-run", false, "Print the command that would run, don't execute")
	Code.Flags().BoolVar(&codeList, "list", false, "List runnable blocks without executing")
	Code.Flags().BoolVar(&codeResults, "results", false, "Write/replace `:down_result` blocks into the file")
	Code.Flags().IntVar(&codeTimeout, "timeout", 0, "Kill a block after N seconds (needs `timeout`)")
	Code.Flags().StringVar(&codeCwd, "cwd", "", "Run blocks in this directory")
	Code.Flags().BoolVar(&codeNoColor, "no-color", false, "Disable color (no-op; output is plain)")
	CodeTangle.Flags().StringVarP(&codeDir, "dir", "d", "", "Tangle output directory (relative paths resolve here)")
	Code.AddCommand(&CodeTangle)
	_ = runtime.GOOS // keep runtime import on platforms that would otherwise drop it
}
