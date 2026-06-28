package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	rootDir := "."
	repoPrefix := "github.com/aamoghS/sideprojects"

	// Find all go.mod files
	type ModInfo struct {
		Path    string
		OldName string
		NewName string
	}
	var mods []ModInfo

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "go.mod" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			lines := strings.Split(string(content), "\n")
			var oldName string
			for _, line := range lines {
				if strings.HasPrefix(line, "module ") {
					oldName = strings.TrimSpace(strings.TrimPrefix(line, "module "))
					break
				}
			}
			if oldName != "" {
				relDir := filepath.ToSlash(filepath.Dir(path))
				var newName string
				if relDir == "." {
					newName = repoPrefix
				} else {
					newName = repoPrefix + "/" + relDir
				}
				mods = append(mods, ModInfo{
					Path:    path,
					OldName: oldName,
					NewName: newName,
				})
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Found modules:")
	for _, m := range mods {
		fmt.Printf("%s: %s -> %s\n", m.Path, m.OldName, m.NewName)
	}

	// Update all go files and go.mod files
	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".go") || d.Name() == "go.mod" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			newContent := string(content)
			changed := false

			if d.Name() == "go.mod" {
				for _, m := range mods {
					// replace module name
					if strings.Contains(newContent, "module "+m.OldName) {
						newContent = strings.Replace(newContent, "module "+m.OldName, "module "+m.NewName, 1)
						changed = true
					}
					// replace require
					if strings.Contains(newContent, " "+m.OldName+" ") {
						newContent = strings.ReplaceAll(newContent, " "+m.OldName+" ", " "+m.NewName+" ")
						changed = true
					}
					if strings.Contains(newContent, "\t"+m.OldName+" ") {
						newContent = strings.ReplaceAll(newContent, "\t"+m.OldName+" ", "\t"+m.NewName+" ")
						changed = true
					}
					// replace replace left side
					if strings.Contains(newContent, "\t"+m.OldName+" =>") {
						newContent = strings.ReplaceAll(newContent, "\t"+m.OldName+" =>", "\t"+m.NewName+" =>")
						changed = true
					}
					if strings.Contains(newContent, " "+m.OldName+" =>") {
						newContent = strings.ReplaceAll(newContent, " "+m.OldName+" =>", " "+m.NewName+" =>")
						changed = true
					}
				}
			} else {
				// .go files
				for _, m := range mods {
					// replace exactly "oldName"
					exact := "\"" + m.OldName + "\""
					if strings.Contains(newContent, exact) {
						newContent = strings.ReplaceAll(newContent, exact, "\""+m.NewName+"\"")
						changed = true
					}
					// replace "oldName/...
					prefix := "\"" + m.OldName + "/"
					if strings.Contains(newContent, prefix) {
						newContent = strings.ReplaceAll(newContent, prefix, "\""+m.NewName+"/")
						changed = true
					}
				}
			}

			if changed {
				fmt.Println("Updating", path)
				os.WriteFile(path, []byte(newContent), 0644)
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
