// Globals written in JS. `host` provides the Go side.
(function (host) {
  function inspect(v, seen = new Set()) {
    if (typeof v === "string") return JSON.stringify(v);
    if (typeof v === "function") return `[Function: ${v.name || "(anonymous)"}]`;
    if (typeof v === "bigint") return `${v}n`;
    if (v === null || typeof v !== "object") return String(v);
    if (v instanceof Error) return v.stack || String(v);
    if (seen.has(v)) return "[Circular]";
    seen.add(v);
    try {
      const list = (xs) => xs.map((x) => inspect(x, seen)).join(", ");
      if (Array.isArray(v)) return `[${list(v)}]`;
      if (v instanceof Map) return `Map(${v.size}) {${[...v].map(([k, x]) => ` ${inspect(k, seen)} => ${inspect(x, seen)}`).join(",")} }`;
      if (v instanceof Set) return `Set(${v.size}) {${v.size ? ` ${list([...v])} ` : ""}}`;
      const body = Object.entries(v).map(([k, x]) => `${k}: ${inspect(x, seen)}`).join(", ");
      return body ? `{ ${body} }` : "{}";
    } finally {
      seen.delete(v);
    }
  }
  const format = (xs) => xs.map((x) => (typeof x === "string" ? x : inspect(x))).join(" ");
  globalThis.console = {
    log: (...xs) => host.stdout(format(xs)),
    error: (...xs) => host.stderr(format(xs)),
  };

  class TextEncoder {
    get encoding() { return "utf-8"; }
    encode(s = "") { return new Uint8Array(host.utf8Encode(String(s))); }
  }
  class TextDecoder {
    get encoding() { return "utf-8"; }
    decode(b) { return b === undefined ? "" : host.utf8Decode(b instanceof ArrayBuffer ? new Uint8Array(b) : b); }
  }
  // goja can't observe GC, so these never collect (which the spec allows).
  class WeakRef {
    #target;
    constructor(target) { this.#target = target; }
    deref() { return this.#target; }
  }
  class FinalizationRegistry {
    register() {}
    unregister() { return false; }
  }
  const queueMicrotask = (fn) => void Promise.resolve().then(fn);
  Object.assign(globalThis, { TextEncoder, TextDecoder, WeakRef, FinalizationRegistry, queueMicrotask });

  return inspect;
})
