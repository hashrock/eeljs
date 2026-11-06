// @eel-allow net api.github.com
// @eel-allow env GITHUB_TOKEN

// 使い方: eel run examples/stars.ts dop251/goja
const repo = eel.args[0] ?? "dop251/goja";
const token = eel.env("GITHUB_TOKEN");

const res = eel.fetch(`https://api.github.com/repos/${repo}`, {
  headers: token ? { Authorization: `Bearer ${token}` } : {},
});
if (res.status !== 200) {
  console.error(`${repo}: ${res.status}`);
  eel.exit(1);
}
const { stargazers_count } = JSON.parse(res.text);
console.log(`${repo}: ★ ${stargazers_count}`);
