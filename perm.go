package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Grant は「このスクリプトに、この種類のアクセスを、この対象へ許可した」という記録。
type Grant struct {
	Kind   string `json:"kind"`   // read | write | net | env | run
	Target string `json:"target"` // パス / ホスト / 変数名 / コマンド名
}

var pinMark = regexp.MustCompile(`(?m)^// @eel-pin\s*$`)

// appID は許可を紐付けるアプリの識別子。
// 通常はパス。エントリに `// @eel-pin` があれば中身のハッシュも含め、書き換えると許可が外れる。
func appID(path, code string) string {
	src, _ := os.ReadFile(path)
	if !pinMark.Match(src) {
		return path
	}
	sum := sha256.Sum256([]byte(code))
	return path + "#" + hex.EncodeToString(sum[:])[:12]
}

// revoke はパスに紐づく許可を、ハッシュ違いも含めてすべて消す。
func (s *Store) revoke(path string) {
	for id := range s.Grants {
		if id == path || strings.HasPrefix(id, path+"#") {
			delete(s.Grants, id)
		}
	}
}

// Store はアプリごとに「常に許可」された Grant を保存する。
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

// covers: パス系はディレクトリ許可が配下を含む。それ以外は完全一致。
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

// Guard は1回の実行中の許可判定を担う。
type Guard struct {
	path    string
	app     string
	store   *Store
	session []Grant // 「今回のみ」許可
	tty     *bufio.ReadWriter
}

func (g *Guard) check(want Grant) error {
	for _, have := range slices.Concat(g.store.Grants[g.app], g.session) {
		if have.covers(want) {
			return nil
		}
	}
	switch g.ask(want) {
	case "a":
		grants := append(g.store.Grants[g.app], want)
		g.store.revoke(g.path) // 古いハッシュの許可を掃除する
		g.store.Grants[g.app] = grants
		if err := g.store.save(); err != nil {
			fmt.Fprintln(os.Stderr, "eel: 許可の保存に失敗:", err)
		}
		return nil
	case "y":
		g.session = append(g.session, want)
		return nil
	}
	return fmt.Errorf("PermissionDenied: %s %s", want.Kind, want.Target)
}

// ask は端末で利用者に尋ねる。端末が無ければ拒否。
func (g *Guard) ask(want Grant) string {
	if g.tty == nil {
		f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return "n"
		}
		g.tty = bufio.NewReadWriter(bufio.NewReader(f), bufio.NewWriter(f))
	}
	fmt.Fprintf(g.tty, "\n⚠️  %s が %s へのアクセスを求めています: %s\n", filepath.Base(g.app), want.Kind, want.Target)
	fmt.Fprint(g.tty, "   [y] 今回のみ  [a] 常に許可  [n] 拒否 > ")
	g.tty.Flush()
	line, _ := g.tty.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(line))
}
