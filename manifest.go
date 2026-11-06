package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Manifest はエントリファイルの `// @eel-...` 行に書かれた宣言。
type Manifest struct {
	Pin   bool    // 中身のハッシュでもアプリを区別する
	Allow []Grant // 実行前にまとめて許可を取る
}

var directive = regexp.MustCompile(`(?m)^// @eel-(pin|allow)\b(.*)$`)

var kinds = []string{"read", "write", "net", "env", "run"}

func readManifest(path string) (Manifest, error) {
	var m Manifest
	src, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	for _, d := range directive.FindAllStringSubmatch(string(src), -1) {
		if d[1] == "pin" {
			m.Pin = true
			continue
		}
		f := strings.Fields(d[2])
		if len(f) != 2 || !slices.Contains(kinds, f[0]) {
			return m, fmt.Errorf("宣言の書式が不正です: %s", d[0])
		}
		g := Grant{f[0], f[1]}
		// 宣言はスクリプトの持ち物なので、相対パスはスクリプトの場所を基準にする
		if (g.Kind == "read" || g.Kind == "write") && !filepath.IsAbs(g.Target) {
			g.Target = filepath.Join(filepath.Dir(path), g.Target)
		}
		m.Allow = append(m.Allow, g)
	}
	return m, nil
}
