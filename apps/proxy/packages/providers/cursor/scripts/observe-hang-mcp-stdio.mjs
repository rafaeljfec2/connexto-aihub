#!/usr/bin/env node
import * as readline from "node:readline";

const rl = readline.createInterface({ input: process.stdin });
let sideEffect = false;

const write = (payload) => {
  try {
    process.stdout.write(`${JSON.stringify(payload)}\n`);
  } catch {
  }
};

rl.on("line", (line) => {
  if (!line.trim()) return;
  let message;
  try {
    message = JSON.parse(line);
  } catch {
    return;
  }
  const { id, method } = message;
  if (method === "initialize") {
    write({
      jsonrpc: "2.0",
      id,
      result: {
        protocolVersion: "2024-11-05",
        capabilities: { tools: {} },
        serverInfo: { name: "observe-hang-mcp", version: "0.0.1" },
      },
    });
    return;
  }
  if (method === "tools/list") {
    write({
      jsonrpc: "2.0",
      id,
      result: {
        tools: [
          {
            name: "observe_ping",
            description: "Hang without executing client work.",
            inputSchema: { type: "object", properties: { ping: { type: "boolean" } } },
          },
        ],
      },
    });
    return;
  }
  if (method === "tools/call") {
    sideEffect = true;
    // Intentionally never respond so the host can cancel a running tool_call.
    return;
  }
});

process.on("exit", () => {
  if (sideEffect) {
    process.stderr.write("observe-hang-mcp: tools/call observed before exit\n");
  }
});
