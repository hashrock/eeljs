# eel

許可したアクセスを覚えてくれる、簡易 CLI 向けの小さな JS/TS ランタイム。

```sh
go build -o eel .
eel run examples/hello.ts
```

スクリプトが初めてアクセスしようとしたとき、Android のように確認します。

```
⚠️  hello.ts が net へのアクセスを求めています: example.com
   [y] 今回のみ  [a] 常に許可  [n] 拒否 >
```

「常に許可」はスクリプトのパスごとに保存されます（`eel perms` / `eel revoke <script>`）。
端末が無い環境では拒否されます。

### 中身で区別する（任意）

エントリファイルに `// @eel-pin` の行があると、パスに加えて中身のハッシュでも区別します。
コード（`import` 先を含む）を書き換えると、許可を出し直すことになります。

## 方針

- **速さより単純さ**: API はすべて同期。イベントループは持たない。
- TS と `import`（相対パス・URL）は esbuild でバンドルし、goja で実行する。
- URL の import は初回だけ取得してキャッシュする（`~/Library/Caches/eel/deps`）。取得に許可は不要。
- 許可の種類: `read` / `write`（ディレクトリを許可すると配下も含む）、`net`（ホスト）、`env`（変数名）、`run`（コマンド名）

API の型定義: `eel.d.ts`
