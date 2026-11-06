import { greet } from "./lib.ts";

const name: string = eel.args[0] ?? "world";
console.log(greet(name));

const text = eel.readTextFile("examples/lib.ts");
console.log(`lib.ts は ${text.split("\n").length} 行`);

const res = eel.fetch("https://example.com");
console.log("example.com:", res.status, res.headers["content-type"]);
