package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Grant records that a script may perform one kind of access on one target.
type Grant struct {
	Kind   string `json:"kind"`   // read | write | net | env | run
	Target string `json:"target"` // path / host / variable name / command name
}

// appID identifies the app that grants belong to.
// Normally the path. When pinned it includes a hash of the code, so editing the code drops the grants.
func appID(path, code string, pin bool) string {
	if !pin {
		return path
	}
	sum := sha256.Sum256([]byte(code))
	return path + "#" + hex.EncodeToString(sum[:])[:12]
}

// revoke removes every grant for the path, including other hashes.
func (s *Store) revoke(path string) {
	for id := range s.Grants {
		if id == path || strings.HasPrefix(id, path+"#") {
			delete(s.Grants, id)
		}
	}
}

// Store persists "always" grants per app.
type Store struct {
	path   string
	Grants map[string][]Grant `json:"grants"`
}

func storePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "eel", "grants.json")
}

func loadStore() *Store {
	s := &Store{path: storePath(), Grants: map[string][]Grant{}}
	if b, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(b, s)
	}
	return s
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(s.path, b, 0o600)
}

// covers: a directory grant covers its contents; other kinds must match exactly.
func (g Grant) covers(want Grant) bool {
	if g.Kind != want.Kind {
		return false
	}
	if g.Kind == "read" || g.Kind == "write" {
		rel, err := filepath.Rel(g.Target, want.Target)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	return g.Target == want.Target
}

// Guard decides permissions during a single run.
type Guard struct {
	path    string
	app     string
	store   *Store
	session []Grant // "this time" grants
	tty     *bufio.ReadWriter
}

// check asks once for everything not yet granted.
func (g *Guard) check(wants ...Grant) error {
	have := slices.Concat(g.store.Grants[g.app], g.session)
	var missing []Grant
	for _, w := range wants {
		if !slices.ContainsFunc(have, func(h Grant) bool { return h.covers(w) }) {
			missing = append(missing, w)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	switch g.ask(missing) {
	case "a":
		grants := append(g.store.Grants[g.app], missing...)
		g.store.revoke(g.path) // drop grants for old hashes
		g.store.Grants[g.app] = grants
		if err := g.store.save(); err != nil {
			fmt.Fprintln(os.Stderr, "eel: failed to save grants:", err)
		}
		return nil
	case "y":
		g.session = append(g.session, missing...)
		return nil
	}
	return fmt.Errorf("PermissionDenied: %s", strings.Join(lines(missing), ", "))
}

func lines(gs []Grant) []string {
	out := make([]string, len(gs))
	for i, g := range gs {
		out[i] = g.Kind + " " + g.Target
	}
	return out
}

// ask prompts on the terminal. Without one, it denies.
func (g *Guard) ask(wants []Grant) string {
	if g.tty == nil {
		f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return "n"
		}
		g.tty = bufio.NewReadWriter(bufio.NewReader(f), bufio.NewWriter(f))
	}
	fmt.Fprintf(g.tty, "\n⚠️  %s requests access to:\n", filepath.Base(g.app))
	for _, w := range wants {
		fmt.Fprintf(g.tty, "     %-5s %s\n", w.Kind, w.Target)
	}
	fmt.Fprint(g.tty, "   [y] this time  [a] always  [n] deny > ")
	g.tty.Flush()
	line, _ := g.tty.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(line))
}
