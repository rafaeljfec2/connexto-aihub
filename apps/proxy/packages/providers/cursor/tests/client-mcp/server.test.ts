import { spawn } from "node:child_process";
import { PassThrough } from "node:stream";
import { describe, expect, it } from "vitest";
import { CLIENT_MCP_SERVER_NAME, FORWARDED_TO_OUTER_CLIENT } from "../../src/client-mcp/constants.js";
import { clientMcpToolDefinitions } from "../../src/client-mcp/definitions.js";
import { runClientForwardingMcpServer } from "../../src/client-mcp/server.js";
import { startClientToolCallbackServer } from "../../src/client-mcp/callback-server.js";
import {
  CLIENT_MCP_CALLBACK_TOKEN_ENV,
  CLIENT_MCP_CALLBACK_URL_ENV,
  CLIENT_MCP_TOOLS_ENV,
  clientForwardingMcpServers,
} from "../../src/client-mcp/forwarding.js";

async function readJsonLines(
  stream: PassThrough,
  count: number,
  timeoutMs = 2000,
): Promise<unknown[]> {
  const lines: unknown[] = [];
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error("Timed out waiting for MCP output")), timeoutMs);
  let buffer = "";
    const onData = (chunk: Buffer | string): void => {
      buffer += chunk.toString();
      while (true) {
        const newline = buffer.indexOf("\n");
        if (newline < 0) break;
        const line = buffer.slice(0, newline).trim();
        buffer = buffer.slice(newline + 1);
        if (!line) continue;
        lines.push(JSON.parse(line) as unknown);
        if (lines.length >= count) {
          clearTimeout(timer);
          stream.off("data", onData);
          resolve(lines);
          return;
        }
      }
    };
    stream.on("data", onData);
  });
}

describe("client MCP forwarding server", () => {
  it("lists client tools and fallback client_shell", async () => {
    const input = new PassThrough();
    const output = new PassThrough();
    const tools = clientMcpToolDefinitions([{ name: "probe_tool", parameters: { type: "object" } }]);
    const runPromise = runClientForwardingMcpServer({
      tools,
      callbackUrl: "",
      callbackToken: "",
      input,
      output,
    });

    input.write(`${JSON.stringify({ jsonrpc: "2.0", id: 1, method: "tools/list" })}\n`);
    const [line] = await readJsonLines(output, 1);
    expect(line).toMatchObject({
      id: 1,
      result: {
        tools: expect.arrayContaining([
          expect.objectContaining({ name: "probe_tool" }),
          expect.objectContaining({ name: "client_shell" }),
        ]),
      },
    });
    input.end();
    await runPromise;
  });

  it("rejects unknown tools and forwards valid tools through the callback", async () => {
    let captured: { toolName: string; arguments: Record<string, unknown> } | undefined;
    const callback = await startClientToolCallbackServer((body) => {
      captured = body;
      return true;
    });
    const input = new PassThrough();
    const output = new PassThrough();
    const tools = clientMcpToolDefinitions([
      {
        name: "write",
        parameters: {
          type: "object",
          properties: { path: { type: "string" }, fileText: { type: "string" } },
          required: ["path", "fileText"],
        },
      },
    ]);
    const runPromise = runClientForwardingMcpServer({
      tools,
      callbackUrl: callback.url,
      callbackToken: callback.token,
      input,
      output,
    });

    input.write(
      `${JSON.stringify({
        jsonrpc: "2.0",
        id: 2,
        method: "tools/call",
        params: { name: "unknown_tool", arguments: {} },
      })}\n`,
    );
    const [errorLine] = await readJsonLines(output, 1);
    expect(errorLine).toMatchObject({ id: 2, error: expect.objectContaining({ message: expect.any(String) }) });

    input.write(
      `${JSON.stringify({
        jsonrpc: "2.0",
        id: 3,
        method: "tools/call",
        params: { name: "write", arguments: { path: "a.txt", fileText: "hi" } },
      })}\n`,
    );
    const [okLine] = await readJsonLines(output, 1);
    expect(okLine).toMatchObject({
      id: 3,
      result: { content: [{ type: "text", text: "FORWARDED_TO_OUTER_CLIENT" }] },
    });
    expect(captured).toEqual({ toolName: "write", arguments: { path: "a.txt", fileText: "hi" } });
    input.end();
    await runPromise;
    await callback.close();
  });

  it("builds mcpServers.client for the SDK", () => {
    const config = clientForwardingMcpServers(
      [{ name: "exec" }],
      { callbackUrl: "http://127.0.0.1:9/callback", callbackToken: "tok" },
    );
    const server = config[CLIENT_MCP_SERVER_NAME] as Record<string, unknown>;
    expect(server.type).toBe("stdio");
    expect(server.command).toBe(process.execPath);
    expect(server.args?.[0]).toBe("--import");
    expect(String(server.args?.[1] ?? "")).toMatch(/tsx[/\\].*esm/i);
    expect(server.args?.[2]).toEqual(expect.stringMatching(/stdio-entry\.ts$/));
    expect(server.env).toMatchObject({
      CURSOR_SDK_BRIDGE_CALLBACK_URL: "http://127.0.0.1:9/callback",
      CURSOR_SDK_BRIDGE_CALLBACK_TOKEN: "tok",
    });
  });

  it("spawns the production stdio-entry.ts path and forwards tools", async () => {
    let captured: { toolName: string; arguments: Record<string, unknown> } | undefined;
    const callback = await startClientToolCallbackServer((body) => {
      captured = body;
      return true;
    });
    const tools = clientMcpToolDefinitions([
      {
        name: "write",
        parameters: {
          type: "object",
          properties: { path: { type: "string" }, fileText: { type: "string" } },
          required: ["path", "fileText"],
        },
      },
    ]);
    const config = clientForwardingMcpServers([{ name: "write" }], {
      callbackUrl: callback.url,
      callbackToken: callback.token,
    });
    const server = config[CLIENT_MCP_SERVER_NAME] as {
      command: string;
      args: string[];
      env: Record<string, string>;
    };
    const child = spawn(server.command, server.args, {
      env: {
        ...process.env,
        [CLIENT_MCP_CALLBACK_URL_ENV]: callback.url,
        [CLIENT_MCP_CALLBACK_TOKEN_ENV]: callback.token,
        [CLIENT_MCP_TOOLS_ENV]: JSON.stringify(tools),
      },
      stdio: ["pipe", "pipe", "pipe"],
    });
    const output = new PassThrough();
    child.stdout?.pipe(output);

    child.stdin?.write(
      `${JSON.stringify({
        jsonrpc: "2.0",
        id: 1,
        method: "tools/call",
        params: { name: "write", arguments: { path: "a.txt", fileText: "hi" } },
      })}\n`,
    );
    const [okLine] = await readJsonLines(output, 1, 8000);
    expect(okLine).toMatchObject({
      id: 1,
      result: { content: [{ type: "text", text: FORWARDED_TO_OUTER_CLIENT }] },
    });
    expect(captured).toEqual({ toolName: "write", arguments: { path: "a.txt", fileText: "hi" } });
    child.stdin?.end();
    await new Promise<void>((resolve) => child.on("close", () => resolve()));
    await callback.close();
  });
});
