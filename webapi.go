package main

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/url"
)

// installWebAPI adds web-standard globals that do no I/O.
func installWebAPI(vm *goja.Runtime) {
	// Enabling require would allow reading files without permission, so initialize the module directly.
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
		Object.assign(globalThis, { TextEncoder, TextDecoder, WeakRef, FinalizationRegistry });
	})`)
	fn, _ := goja.AssertFunction(setup)
	fn(goja.Undefined(),
		vm.ToValue(func(s string) goja.ArrayBuffer { return vm.NewArrayBuffer([]byte(s)) }),
		vm.ToValue(func(b []byte) string { return string(b) }))
}
