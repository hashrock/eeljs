package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
)

// bundle combines TS/JS and its imports into a single JS file.
func bundle(entry string) (string, error) {
	r := api.Build(api.BuildOptions{
		EntryPoints: []string{entry},
		Bundle:      true,
		Write:       false,
		Format:      api.FormatIIFE,
		Target:      api.ES2017,
		Platform:    api.PlatformNeutral,
		LogLevel:    api.LogLevelError,
		Plugins:     []api.Plugin{remoteImports},
	})
	if len(r.Errors) > 0 {
		return "", fmt.Errorf("%s", r.Errors[0].Text)
	}
	return string(r.OutputFiles[0].Contents), nil
}

type exitCode int

func run(script string, args []string) int {
	path, _ := filepath.Abs(script)
	code, err := bundle(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "eel:", err)
		return 1
	}

	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	m, err := readManifest(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "eel:", err)
		return 1
	}
	guard := &Guard{path: path, app: appID(path, code, m.Pin), store: loadStore()}
	// Declared permissions are required, so don't run without them.
	if err := guard.check(m.Allow...); err != nil {
		fmt.Fprintln(os.Stderr, "eel:", err)
		return 1
	}
	l := newLoop(vm, installWebAPI(vm))
	install(vm, guard, args)

	var status int
	func() {
		defer func() {
			if r := recover(); r != nil {
				c, ok := r.(exitCode)
				if !ok {
					panic(r)
				}
				status = int(c)
			}
		}()
		if err := l.run(code); err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
		}
	}()
	return status
}

type fetchResult struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Text    string            `json:"text"`
}

type fetchInit struct {
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type runResult struct {
	Code   int    `json:"code"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

// install provides synchronous APIs only, so the event loop only has timers to wait on.
func install(vm *goja.Runtime, g *Guard, args []string) {
	// must checks the permission and throws a JS exception if denied.
	must := func(kind, target string) {
		if err := g.check(Grant{kind, target}); err != nil {
			panic(vm.NewGoError(err))
		}
	}
	abs := func(p string) string { a, _ := filepath.Abs(p); return a }
	try := func(err error) {
		if err != nil {
			panic(vm.NewGoError(err))
		}
	}

	eel := vm.NewObject()
	eel.Set("args", args)
	eel.Set("exit", func(code int) { panic(exitCode(code)) })

	eel.Set("readTextFile", func(path string) string {
		p := abs(path)
		must("read", p)
		b, err := os.ReadFile(p)
		try(err)
		return string(b)
	})
	eel.Set("writeTextFile", func(path, text string) {
		p := abs(path)
		must("write", p)
		try(os.WriteFile(p, []byte(text), 0o644))
	})
	eel.Set("env", func(name string) any {
		must("env", name)
		v, ok := os.LookupEnv(name)
		if !ok {
			return nil
		}
		return v
	})
	eel.Set("fetch", func(rawURL string, init *fetchInit) fetchResult {
		u, err := url.Parse(rawURL)
		try(err)
		must("net", u.Host)
		if init == nil {
			init = &fetchInit{}
		}
		if init.Method == "" {
			init.Method = "GET"
		}
		req, err := http.NewRequest(init.Method, rawURL, strings.NewReader(init.Body))
		try(err)
		for k, v := range init.Headers {
			req.Header.Set(k, v)
		}
		res, err := http.DefaultClient.Do(req)
		try(err)
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		try(err)
		h := map[string]string{}
		for k := range res.Header {
			h[strings.ToLower(k)] = res.Header.Get(k)
		}
		return fetchResult{res.StatusCode, h, string(body)}
	})
	eel.Set("run", func(cmd string, cmdArgs []string) runResult {
		must("run", cmd)
		var out, errOut bytes.Buffer
		c := exec.Command(cmd, cmdArgs...)
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, &out, &errOut
		err := c.Run()
		if _, ok := err.(*exec.ExitError); err != nil && !ok {
			try(err)
		}
		return runResult{c.ProcessState.ExitCode(), out.String(), errOut.String()}
	})
	vm.Set("eel", eel)
}
