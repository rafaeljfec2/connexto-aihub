import * as readline from "node:readline";
import { FORWARDED_TO_OUTER_CLIENT } from "./constants.js";
import type { ClientMcpToolDefinition } from "./definitions.js";
import { validateClientMcpToolCall } from "./validate.js";

export interface ClientMcpServerOptions {
  readonly tools: readonly ClientMcpToolDefinition[];
  readonly callbackUrl: string;
  readonly callbackToken: string;
  readonly input?: NodeJS.ReadableStream;
  readonly output?: NodeJS.WritableStream;
}

export async function runClientForwardingMcpServer(options: ClientMcpServerOptions): Promise<void> {
  const input = options.input ?? process.stdin;
  const output = options.output ?? process.stdout;
  const rl = readline.createInterface({ input });
  let outputClosed = false;
  const writeOutput = (payload: string): boolean => {
    if (outputClosed) return false;
    try {
      return output.write(payload);
    } catch {
      outputClosed = true;
      return false;
    }
  };
  output.on?.("error", () => {
    outputClosed = true;
  });

  const send = (id: unknown, result: unknown): void => {
    if (id === undefined || id === null) return;
    writeOutput(`${JSON.stringify({ jsonrpc: "2.0", id, result })}\n`);
  };
  const sendError = (id: unknown, message: string): void => {
    if (id === undefined || id === null) return;
    writeOutput(
      `${JSON.stringify({ jsonrpc: "2.0", id, error: { code: -32_000, message } })}\n`,
    );
  };

  const pending = new Set<Promise<void>>();

  rl.on("line", (line) => {
    const task = (async () => {
      if (!line.trim()) return;
      let message: Record<string, unknown>;
      try {
        message = JSON.parse(line) as Record<string, unknown>;
      } catch {
        return;
      }
      const method = String(message.method ?? "");
      if (!message.id && method.startsWith("notifications/")) return;
      if (method === "initialize") {
        send(message.id, {
          protocolVersion: "2024-11-05",
          capabilities: { tools: {} },
          serverInfo: { name: "api-for-cursor-client-tools", version: "0.1.0" },
        });
        return;
      }
      if (method === "tools/list") {
        send(message.id, { tools: options.tools });
        return;
      }
      if (method === "tools/call") {
        const params = (message.params ?? {}) as Record<string, unknown>;
        const toolName = String(params.name ?? params.toolName ?? "");
        const toolInput = (params.arguments ?? params.input ?? {}) as unknown;
        const validationError = validateClientMcpToolCall(options.tools, toolName, toolInput);
        if (validationError) {
          sendError(message.id, validationError);
          return;
        }
        const accepted = await notifyParentToolCall(
          options.callbackUrl,
          options.callbackToken,
          toolName,
          toolInput,
        );
        if (!accepted) {
          sendError(message.id, "Outer client callback unavailable for forwarded tool call.");
          return;
        }
        send(message.id, {
          content: [{ type: "text", text: FORWARDED_TO_OUTER_CLIENT }],
          isError: false,
        });
        return;
      }
      sendError(message.id, `Unsupported MCP method: ${method}`);
    })();
    pending.add(task);
    void task.finally(() => {
      pending.delete(task);
    });
  });

  await new Promise<void>((resolve) => {
    rl.on("close", async () => {
      await Promise.allSettled([...pending]);
      resolve();
    });
  });
}

async function notifyParentToolCall(
  callbackUrl: string,
  callbackToken: string,
  toolName: string,
  input: unknown,
): Promise<boolean> {
  if (!callbackUrl.trim()) return true;
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 1500);
  try {
    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (callbackToken) headers.Authorization = `Bearer ${callbackToken}`;
    const args =
      input !== null && typeof input === "object" && !Array.isArray(input)
        ? (input as Record<string, unknown>)
        : {};
    const response = await fetch(callbackUrl, {
      method: "POST",
      headers,
      body: JSON.stringify({ toolName, arguments: args }),
      signal: controller.signal,
    });
    if (!response.ok) return false;
    const body = (await response.json().catch(() => ({}))) as { accepted?: boolean };
    return body?.accepted === true;
  } catch {
    return false;
  } finally {
    clearTimeout(timer);
  }
}
