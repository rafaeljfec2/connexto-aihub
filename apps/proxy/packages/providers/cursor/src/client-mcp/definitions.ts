import type { CursorClientToolSpec } from "@agent-runtime-gateway/core";
import { isRecord } from "@agent-runtime-gateway/compatibility-openai";

export interface ClientMcpToolDefinition {
  readonly name: string;
  readonly description: string;
  readonly inputSchema: Record<string, unknown>;
}

const pathProperty = {
  type: "string",
  description: "File or directory path for the outer client.",
};

const fallbackTools: ClientMcpToolDefinition[] = [
  {
    name: "client_shell",
    description: "Forward a shell command to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        command: { type: "string" },
        workingDirectory: { type: "string" },
        timeout: { type: "number" },
      },
      required: ["command"],
      additionalProperties: true,
    },
  },
  {
    name: "client_write",
    description: "Forward a file write to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        path: pathProperty,
        fileText: { type: "string" },
      },
      required: ["path", "fileText"],
      additionalProperties: true,
    },
  },
  {
    name: "client_read",
    description: "Forward a file read to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        path: pathProperty,
        offset: { type: "number" },
        limit: { type: "number" },
      },
      required: ["path"],
      additionalProperties: true,
    },
  },
  {
    name: "client_edit",
    description: "Forward a text replacement edit to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        path: pathProperty,
        oldString: { type: "string" },
        newString: { type: "string" },
      },
      required: ["path", "oldString", "newString"],
      additionalProperties: true,
    },
  },
  {
    name: "client_delete",
    description: "Forward a file or directory delete to the outer client.",
    inputSchema: {
      type: "object",
      properties: { path: pathProperty },
      required: ["path"],
      additionalProperties: true,
    },
  },
  {
    name: "client_glob",
    description: "Forward a glob file search to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        targetDirectory: { type: "string" },
        globPattern: { type: "string" },
      },
      required: ["globPattern"],
      additionalProperties: true,
    },
  },
  {
    name: "client_grep",
    description: "Forward a text search to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        pattern: { type: "string" },
        path: { type: "string" },
        glob: { type: "string" },
      },
      required: ["pattern"],
      additionalProperties: true,
    },
  },
  {
    name: "client_ls",
    description: "Forward a directory listing to the outer client.",
    inputSchema: {
      type: "object",
      properties: { path: { type: "string" } },
      additionalProperties: true,
    },
  },
  {
    name: "client_read_lints",
    description: "Forward diagnostics/lint reads to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        paths: { type: "array", items: { type: "string" } },
      },
      required: ["paths"],
      additionalProperties: true,
    },
  },
  {
    name: "client_sem_search",
    description: "Forward semantic code search to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        query: { type: "string" },
        targetDirectories: { type: "array", items: { type: "string" } },
      },
      required: ["query"],
      additionalProperties: true,
    },
  },
  {
    name: "client_todo_write",
    description: "Forward todo list updates to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        todos: { type: "array", items: { type: "object", additionalProperties: true } },
      },
      required: ["todos"],
      additionalProperties: true,
    },
  },
  {
    name: "client_task",
    description: "Forward a subagent/task request to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        description: { type: "string" },
        prompt: { type: "string" },
      },
      required: ["description", "prompt"],
      additionalProperties: true,
    },
  },
  {
    name: "client_create_plan",
    description: "Forward a plan creation/update request to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        plan: { type: "string" },
        overview: { type: "string" },
        name: { type: "string" },
      },
      additionalProperties: true,
    },
  },
  {
    name: "client_generate_image",
    description: "Forward an image generation request to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        description: { type: "string" },
        filePath: pathProperty,
      },
      required: ["description"],
      additionalProperties: true,
    },
  },
  {
    name: "client_record_screen",
    description: "Forward a screen recording control request to the outer client.",
    inputSchema: {
      type: "object",
      properties: {
        mode: {
          type: "string",
          enum: ["START_RECORDING", "SAVE_RECORDING", "DISCARD_RECORDING"],
        },
      },
      required: ["mode"],
      additionalProperties: true,
    },
  },
];

export function clientMcpToolDefinitions(
  clientTools: readonly CursorClientToolSpec[] = [],
): ClientMcpToolDefinition[] {
  const tools: ClientMcpToolDefinition[] = [];
  const seen = new Set<string>();
  for (const tool of clientTools) {
    const name = tool.name.trim();
    if (!name || seen.has(name)) continue;
    seen.add(name);
    tools.push({
      name,
      description: tool.description ?? `Forward ${name} to the outer client.`,
      inputSchema:
        tool.inputFormat === "grammar"
          ? grammarToolInputSchema()
          : clientMcpInputSchema(tool.parameters),
    });
  }
  for (const tool of fallbackTools) {
    if (seen.has(tool.name)) continue;
    seen.add(tool.name);
    tools.push(tool);
  }
  return tools;
}

export function grammarToolInputSchema(): Record<string, unknown> {
  return {
    type: "object",
    properties: {
      input: {
        type: "string",
        description:
          "Raw custom tool input. This string is forwarded unchanged as the OpenAI custom_tool_call input.",
      },
    },
    required: ["input"],
    additionalProperties: false,
  };
}

export function clientMcpInputSchema(parameters: unknown): Record<string, unknown> {
  if (!isRecord(parameters)) {
    return { type: "object", additionalProperties: true };
  }
  return { ...parameters, type: parameters.type ?? "object" };
}

export function parseClientMcpToolsFromEnv(json: string | undefined): ClientMcpToolDefinition[] {
  if (typeof json !== "string" || !json.trim()) return clientMcpToolDefinitions([]);
  try {
    const parsed: unknown = JSON.parse(json);
    if (!Array.isArray(parsed)) return clientMcpToolDefinitions([]);
    return parsed.filter(
      (item): item is ClientMcpToolDefinition =>
        isRecord(item) && typeof item.name === "string" && isRecord(item.inputSchema),
    );
  } catch {
    return clientMcpToolDefinitions([]);
  }
}
