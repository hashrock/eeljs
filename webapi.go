package main

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/url"
)

// installWebAPI は I/O を伴わない Web 標準のグローバルを足す。
func installWebAPI(vm *goja.Runtime) {
	// require を有効にすると許可を経ずにファイルを読めるため、モジュールを直接初期化する
	mod := vm.NewObject()
	exports := vm.NewObject()
	mod.Set("exports", exports)
	url.Require(vm, mod)
	vm.Set("URL", exports.Get("URL"))
	vm.Set("URLSearchParams", exports.Get("URLSearchParams"))

	setup, _ := vm.RunString(`(function (utf8Encode, utf8Decode) {
		class TextEncoder {
			get encoding() { return "utf-8"; }
			encode(s = "") { return new Uint8Array(utf8Encode(String(s))); }
		}
		class TextDecoder {
			get encoding() { return "utf-8"; }
			decode(b) { return b === undefined ? "" : utf8Decode(b instanceof ArrayBuffer ? new Uint8Array(b) : b); }
		}
		// goja に GC 連動の仕組みが無いため、一度も回収しない実装にする（仕様上は許される）
		class WeakRef {
			#target;
			constructor(target) { this.#target = target; }
			deref() { return this.#target; }
		}
		class FinalizationRegistry {
			register() {}
			unregister() { return false; }
		}
		Object.assign(globalThis, { TextEncoder, TextDecoder, WeakRef, FinalizationRegistry });
	})`)
	fn, _ := goja.AssertFunction(setup)
	fn(goja.Undefined(),
		vm.ToValue(func(s string) goja.ArrayBuffer { return vm.NewArrayBuffer([]byte(s)) }),
		vm.ToValue(func(b []byte) string { return string(b) }))
}
