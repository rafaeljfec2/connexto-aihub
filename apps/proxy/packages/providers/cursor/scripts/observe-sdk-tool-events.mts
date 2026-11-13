/**
 * Standalone Cursor SDK 1.3.1 tool-event probe. Does not touch the hub runtime.
 * Records message shapes and correlation ids without persisting tool argument bodies.
 */
import { mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { Agent, type SDKMessage } from "@cursor/sdk";

const __dirname = dirname(fileURLToPath(import.meta.url));

interface SanitizedRecord {
  scenario: string;
  type: string;
  name?: string;
  status?: string;
  call_id?: string;
  tool_use_id?: string;
  argsKeys?: string[];
  resultKeys?: string[];
  customToolCallId?: string;
  deltaTypes?: string[];
  cancelRequested?: boolean;
  messagesAfterCancel?: number;
  notes?: string[];
}

function sanitizeMessage(
  scenario: string,
  message: SDKMessage,
  extras: Partial<SanitizedRecord> = {},
): SanitizedRecord {
  const base: SanitizedRecord = { scenario, type: message.type, ...extras };
  if (message.type === "tool_call") {
    base.name = message.name;
    base.status = message.status;
    base.call_id = message.call_id;
    const args = message.args;
    if (args && typeof args === "object" && !Array.isArray(args)) {
      base.argsKeys = Object.keys(args as Record<string, unknown>).sort();
    }
    const result = message.result;
    if (result && typeof result === "object" && !Array.isArray(result)) {
      base.resultKeys = Object.keys(result as Record<string, unknown>).sort();
    }
  }
  if (message.type === "assistant") {
    for (const block of message.message.content) {
      if (block.type === "tool_use") {
        return {
          ...base,
          type: "assistant.tool_use",
          name: block.name,
          tool_use_id: block.id,
          argsKeys:
            block.input && typeof block.input === "object" && !Array.isArray(block.input)
              ? Object.keys(block.input as Record<string, unknown>).sort()
              : undefined,
        };
      }
    }
  }
  return base;
}

async function drainScenario(input: {
  scenario: string;
  tools: string[];
  prompt: string;
  cwd: string;
  mcpServers?: Record<string, unknown>;
  local?: Record<string, unknown>;
  onRunningToolCall?: (message: SDKMessage, run: { cancel: () => Promise<void> }) => Promise<void>;
}): Promise<SanitizedRecord[]> {
  const records: SanitizedRecord[] = [];
  const agent = await Agent.create({
    apiKey: process.env.CURSOR_API_KEY,
    model: { id: "composer-2.5" },
    name: `observe-${input.scenario}`,
    mode: "agent",
    tools: input.tools,
    mcpServers: input.mcpServers ?? {},
    local: {
      cwd: input.cwd,
      settingSources: [],
      ...(input.local ?? {}),
    },
  });
  const deltaTypes = new Set<string>();
  try {
    const run = await agent.send(
      { text: input.prompt },
      {
        onDelta: ({ update }) => {
          const kind = (update as { type?: string }).type;
          if (kind) deltaTypes.add(kind);
        },
      },
    );
    let cancelRequested = false;
    let messagesAfterCancel = 0;
    for await (const message of run.stream()) {
      if (cancelRequested) messagesAfterCancel += 1;
      if (
        input.onRunningToolCall &&
        message.type === "tool_call" &&
        message.status === "running"
      ) {
        cancelRequested = true;
        await input.onRunningToolCall(message, run);
      }
      records.push(
        sanitizeMessage(input.scenario, message, {
          cancelRequested: cancelRequested || undefined,
        }),
      );
    }
    const result = await run.wait();
    records.push({
      scenario: input.scenario,
      type: "run.result",
      status: result.status,
      deltaTypes: [...deltaTypes].sort(),
      messagesAfterCancel: cancelRequested ? messagesAfterCancel : undefined,
    });
  } finally {
    agent.close();
    await Agent.delete(agent.agentId, { cwd: input.cwd });
  }
  return records;
}

function hangMcpStdioServerPath(): string {
  return join(__dirname, "observe-hang-mcp-stdio.mjs");
}

async function main(): Promise<void> {
  const apiKey = process.env.CURSOR_API_KEY?.trim();
  if (!apiKey) {
    console.error("CURSOR_API_KEY is required");
    process.exit(1);
  }

  const cwd = await mkdtemp(join(tmpdir(), "cursor-sdk-observe-"));
  await writeFile(join(cwd, "probe.txt"), "probe_read_ok\n", "utf8");

  const customToolCallIds: string[] = [];
  const all: SanitizedRecord[] = [];

  all.push(
    ...(await drainScenario({
      scenario: "read",
      tools: ["read"],
      cwd,
      prompt:
        'You must use the read tool on the file probe.txt in the workspace root, then reply with exactly: probe_read_ok',
    })),
  );

  all.push(
    ...(await drainScenario({
      scenario: "shell",
      tools: ["shell"],
      cwd,
      prompt:
        'You must run the shell tool once with command: printf probe_shell_ok\\n — do not use any other tool.',
    })),
  );

  const hangPath = hangMcpStdioServerPath();
  all.push(
    ...(await drainScenario({
      scenario: "mcp_hang_cancel",
      tools: ["mcp"],
      cwd,
      mcpServers: {
        observe_hang: {
          type: "stdio",
          command: process.execPath,
          args: [hangPath],
        },
      },
      prompt:
        'Call the MCP tool named observe_ping once with {"ping":true} via the observe_hang server, then stop.',
      onRunningToolCall: async (_message, run) => {
        await run.cancel();
      },
    })),
  );

  all.push(
    ...(await drainScenario({
      scenario: "customTools",
      tools: ["mcp"],
      cwd,
      local: {
        customTools: {
          observe_probe: {
            description: "No-op probe tool for correlation id capture.",
            parameters: { type: "object", properties: {} },
            execute: (_args, context) => {
              if (context.toolCallId) customToolCallIds.push(context.toolCallId);
              return "probe_custom_ok";
            },
          },
        },
      },
      prompt: "Invoke the observe_probe tool exactly once, then reply: probe_custom_ok",
    })),
  );

  for (const record of all) {
    if (record.type === "tool_call" && record.scenario === "customTools") {
      record.customToolCallId = customToolCallIds.at(-1);
      record.notes = [
        ...(record.notes ?? []),
        "Compare tool_call.call_id, assistant.tool_use.id, and SDKCustomToolContext.toolCallId.",
      ];
    }
  }

  const sdkTypes = {
    sendOnDelta: true,
    partialToolCallLiteral: "partial-tool-call",
    toolCallDeltaLiteral: "tool-call-delta",
    observedDeltaTypes: [
      ...new Set(
        all.flatMap((row) => row.deltaTypes ?? []),
      ),
    ].sort(),
  };

  const output = {
    sdkVersion: "1.3.1",
    workspace: cwd,
    customToolCallIds,
    sdkTypes,
    records: all,
  };

  console.log(JSON.stringify(output, null, 2));
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
