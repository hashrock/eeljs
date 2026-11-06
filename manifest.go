package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Manifest holds the `// @eel-...` declarations in the entry file.
type Manifest struct {
	Pin   bool    // also identify the app by a hash of its code
	Allow []Grant // granted all at once before running
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
			return m, fmt.Errorf("invalid declaration: %s", d[0])
		}
		g := Grant{f[0], f[1]}
		// Declarations belong to the script, so relative paths are resolved from its location.
		if (g.Kind == "read" || g.Kind == "write") && !filepath.IsAbs(g.Target) {
			g.Target = filepath.Join(filepath.Dir(path), g.Target)
		}
		m.Allow = append(m.Allow, g)
	}
	return m, nil
}
