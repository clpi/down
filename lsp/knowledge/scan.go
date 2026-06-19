package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var mdExtensions = map[string]bool{
	".md":       true,
	".markdown": true,
	".mdx":      true,
	".txt":      true,
}

func ScanWorkspace(g *Graph, roots []string) int {
	var mu sync.Mutex
	var wg sync.WaitGroup
	count := 0

	for _, root := range roots {
		root = cleanURI(root)
		wg.Add(1)
		go func(root string) {
			defer wg.Done()
			filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					base := filepath.Base(path)
					if base == ".git" || base == "node_modules" || base == ".obsidian" || base == ".trash" {
						return filepath.SkipDir
					}
					return nil
				}
				ext := strings.ToLower(filepath.Ext(path))
				if !mdExtensions[ext] {
					return nil
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				uri := "file://" + path
				ExtractFromDocument(g, uri, string(data))
				mu.Lock()
				count++
				mu.Unlock()
				return nil
			})
		}(root)
	}
	wg.Wait()
	g.Save()
	return count
}

func cleanURI(uri string) string {
	uri = strings.TrimPrefix(uri, "file://")
	uri = strings.TrimPrefix(uri, "file:")
	return uri
}
