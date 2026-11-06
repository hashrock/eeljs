package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const usage = `eel - a tiny JS/TS runtime that remembers permissions

  eel run <script> [args...]   run a script
  eel perms                    list "always" grants
  eel revoke <script>          revoke all grants for a script
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	switch cmd, rest := os.Args[1], os.Args[2:]; {
	case cmd == "run" && len(rest) > 0:
		os.Exit(run(rest[0], rest[1:]))
	case cmd == "perms":
		listPerms()
	case cmd == "revoke" && len(rest) == 1:
		s := loadStore()
		path, _ := filepath.Abs(rest[0])
		s.revoke(path)
		if err := s.save(); err != nil {
			fmt.Fprintln(os.Stderr, "eel:", err)
			os.Exit(1)
		}
	default:
		fmt.Print(usage)
		os.Exit(2)
	}
}

func listPerms() {
	s := loadStore()
	apps := make([]string, 0, len(s.Grants))
	for app := range s.Grants {
		apps = append(apps, app)
	}
	sort.Strings(apps)
	for _, app := range apps {
		fmt.Println(app)
		for _, g := range s.Grants[app] {
			fmt.Printf("  %-5s %s\n", g.Kind, g.Target)
		}
	}
}
