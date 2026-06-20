package wsutil

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

var MarkdownExtensions = map[string]bool{
	".md":       true,
	".markdown": true,
	".mdx":      true,
	".txt":      true,
}

func ResolveRoot(start string) string {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "."
		}
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		abs = start
	}
	for dir := abs; dir != "" && dir != string(os.PathSeparator); {
		if info, err := os.Stat(filepath.Join(dir, ".down")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return abs
}

func FindDownDir(root string) string {
	root = ResolveRoot(root)
	p := filepath.Join(root, ".down")
	if info, err := os.Stat(p); err == nil && info.IsDir() {
		return p
	}
	return ""
}

func WalkMarkdown(root string, skipDown bool) ([]string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var files []string
	err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			switch filepath.Base(path) {
			case ".git", "node_modules", ".obsidian", ".trash":
				return filepath.SkipDir
			case ".down":
				if skipDown {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !MarkdownExtensions[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

func NoteRelPath(t time.Time, strategy string) string {
	switch strategy {
	case "flat":
		return t.Format("2006-01-02") + ".md"
	default:
		return filepath.Join(t.Format("2006"), t.Format("01"), t.Format("02")+".md")
	}
}

func EnsureNoteAt(root string, t time.Time, strategy, templatePath string) (string, error) {
	root = ResolveRoot(root)
	noteDir := filepath.Join(root, "note")
	rel := NoteRelPath(t, strategy)
	full := filepath.Join(noteDir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return "", err
	}
	if _, err := os.Stat(full); os.IsNotExist(err) {
		content := "# " + t.Format("2006-01-02") + "\n\n"
		if templatePath != "" {
			if data, rerr := os.ReadFile(templatePath); rerr == nil {
				content = strings.ReplaceAll(string(data), "{{date}}", t.Format("2006-01-02"))
				content = strings.ReplaceAll(content, "{{time}}", t.Format("15:04:05"))
			}
		}
		if werr := os.WriteFile(full, []byte(content), 0644); werr != nil {
			return "", werr
		}
	}
	return full, nil
}

func NotePathFor(root string, t time.Time, strategy string) string {
	root = ResolveRoot(root)
	return filepath.Join(root, "note", NoteRelPath(t, strategy))
}
