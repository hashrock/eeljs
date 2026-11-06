# eel

A tiny JS/TS runtime for simple CLIs that remembers the access you've granted.

```sh
go build -o eel .
eel run examples/hello.ts
```

The first time a script tries to access something, eel asks, Android-style.

```
⚠️  hello.ts requests access to:
     net   example.com
   [y] this time  [a] always  [n] deny >
```

"Always" grants are saved per script path (`eel perms` / `eel revoke <script>`).
Without a terminal, access is denied.

### Ask up front (optional)

Declare permissions in the entry file, and eel asks for all ungranted ones once before running. If denied, the script doesn't run.

```ts
// @eel-allow read ./data
// @eel-allow net api.github.com
```

Relative paths are resolved from the script's location. Undeclared access is still asked for at runtime.

### Pin to contents (optional)

With a `// @eel-pin` line in the entry file, grants are tied to a hash of the code as well as its path.
Changing the code (including imports) requires granting again.

## Design

- **Simple over fast**: every I/O API is synchronous. The event loop only runs timers.
- TS and `import` (relative paths and URLs) are bundled with esbuild and run on goja.
- URL imports are fetched once and cached (`~/Library/Caches/eel/deps`). Fetching them needs no permission.
- Web globals: `console` / `setTimeout` / `clearTimeout` / `queueMicrotask` / `URL` / `URLSearchParams` / `TextEncoder` / `TextDecoder` / `WeakRef` / `FinalizationRegistry`
- Permission kinds: `read` / `write` (a directory grant covers its contents), `net` (host), `env` (variable name), `run` (command name)

API types: `eel.d.ts`

## Limitations

- `WeakRef` never releases its target, and `FinalizationRegistry` callbacks never run. goja doesn't support them, so these are stand-ins; this shouldn't matter for short-lived CLIs.
