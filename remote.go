package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
)

// remoteImports は URL からの import を取得してバンドルに含める。
// import のための取得には許可を求めない。
var remoteImports = api.Plugin{
	Name: "url",
	Setup: func(b api.PluginBuild) {
		b.OnResolve(api.OnResolveOptions{Filter: `^https?://`}, func(a api.OnResolveArgs) (api.OnResolveResult, error) {
			return api.OnResolveResult{Path: a.Path, Namespace: "url"}, nil
		})
		// URL モジュール内の import は、取り込み元の URL を基準に解決する
		b.OnResolve(api.OnResolveOptions{Filter: `.*`, Namespace: "url"}, func(a api.OnResolveArgs) (api.OnResolveResult, error) {
			base, err := url.Parse(a.Importer)
			if err != nil {
				return api.OnResolveResult{}, err
			}
			ref, err := url.Parse(a.Path)
			if err != nil {
				return api.OnResolveResult{}, err
			}
			return api.OnResolveResult{Path: base.ResolveReference(ref).String(), Namespace: "url"}, nil
		})
		b.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "url"}, func(a api.OnLoadArgs) (api.OnLoadResult, error) {
			src, err := fetchCached(a.Path)
			if err != nil {
				return api.OnLoadResult{}, err
			}
			return api.OnLoadResult{Contents: &src, Loader: loaderFor(a.Path)}, nil
		})
	},
}

// fetchCached は URL の中身を返す。URL は不変とみなし、一度取得したら再取得しない。
func fetchCached(u string) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(u))
	file := filepath.Join(dir, "eel", "deps", hex.EncodeToString(sum[:]))
	if b, err := os.ReadFile(file); err == nil {
		return string(b), nil
	}

	fmt.Fprintln(os.Stderr, "Download", u)
	res, err := http.Get(u)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: %s", u, res.Status)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", err
	}
	return string(b), os.WriteFile(file, b, 0o644)
}

func loaderFor(u string) api.Loader {
	p, _ := url.Parse(u)
	switch path.Ext(p.Path) {
	case ".ts", ".mts":
		return api.LoaderTS
	case ".tsx":
		return api.LoaderTSX
	case ".jsx":
		return api.LoaderJSX
	case ".json":
		return api.LoaderJSON
	}
	return api.LoaderJS
}
