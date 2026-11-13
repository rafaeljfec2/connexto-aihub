import { randomBytes } from "node:crypto";
import { createServer, type Server } from "node:http";
import { CLIENT_TOOL_CALLBACK_PATH } from "./constants.js";

export interface ClientToolCallbackBody {
  readonly toolName: string;
  readonly arguments: Record<string, unknown>;
}

export interface ClientToolCallbackServer {
  readonly url: string;
  readonly token: string;
  close(): Promise<void>;
}

export function startClientToolCallbackServer(
  onToolCall: (body: ClientToolCallbackBody) => boolean,
): Promise<ClientToolCallbackServer> {
  const token = randomToken();
  let server: Server | undefined;

  return new Promise((resolve, reject) => {
    server = createServer((request, response) => {
      void handleRequest(request, response, token, onToolCall).catch(() => {
        response.statusCode = 500;
        response.end();
      });
    });
    server.on("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server?.address();
      if (
        address === null ||
        address === undefined ||
        typeof address === "string"
      ) {
        reject(new Error("Failed to bind client tool callback server."));
        return;
      }
      const url = `http://127.0.0.1:${address.port}${CLIENT_TOOL_CALLBACK_PATH}`;
      resolve({
        url,
        token,
        close: () => closeServer(server),
      });
    });
  });
}

async function handleRequest(
  request: import("node:http").IncomingMessage,
  response: import("node:http").ServerResponse,
  token: string,
  onToolCall: (body: ClientToolCallbackBody) => boolean,
): Promise<void> {
  const method = request.method ?? "GET";
  const url = new URL(request.url ?? "/", "http://127.0.0.1");
  if (method !== "POST" || url.pathname !== CLIENT_TOOL_CALLBACK_PATH) {
    response.statusCode = 404;
    response.end();
    return;
  }
  const auth = request.headers.authorization ?? "";
  if (auth !== `Bearer ${token}`) {
    response.statusCode = 401;
    response.end();
    return;
  }
  const chunks: Buffer[] = [];
  for await (const chunk of request) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  let body: unknown;
  try {
    body = JSON.parse(Buffer.concat(chunks).toString("utf8")) as unknown;
  } catch {
    response.statusCode = 400;
    response.end();
    return;
  }
  if (
    body === null ||
    typeof body !== "object" ||
    !("toolName" in body) ||
    typeof (body as { toolName?: unknown }).toolName !== "string"
  ) {
    response.statusCode = 400;
    response.end();
    return;
  }
  const record = body as { toolName: string; arguments?: unknown };
  const args =
    record.arguments !== null &&
    typeof record.arguments === "object" &&
    !Array.isArray(record.arguments)
      ? (record.arguments as Record<string, unknown>)
      : {};
  const accepted = onToolCall({ toolName: record.toolName, arguments: args });
  response.setHeader("Content-Type", "application/json");
  response.statusCode = accepted ? 200 : 503;
  response.end(JSON.stringify({ accepted }));
}

function randomToken(): string {
  return `ctc_${randomBytes(24).toString("hex")}`;
}

function closeServer(server: Server | undefined): Promise<void> {
  return new Promise((resolve, reject) => {
    if (server === undefined) {
      resolve();
      return;
    }
    server.close((error) => {
      if (error) reject(error);
      else resolve();
    });
  });
}
