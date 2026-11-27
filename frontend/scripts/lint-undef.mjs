#!/usr/bin/env node
// Fails (exit 1) if ESLint reports any no-undef error across the project.
// Other lint rules are not gated here; they are reported by `npm run lint`.
import { ESLint } from "eslint";

const eslint = new ESLint();
const results = await eslint.lintFiles(["."]);

let count = 0;
for (const file of results) {
  for (const msg of file.messages) {
    if (msg.ruleId !== "no-undef") continue;
    count += 1;
    console.error(`${file.filePath}:${msg.line}:${msg.column} ${msg.message}`);
  }
}

if (count > 0) {
  console.error(`\n${count} no-undef error(s) found.`);
  process.exit(1);
}
console.log("No no-undef errors.");
