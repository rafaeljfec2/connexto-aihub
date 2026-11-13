import type { ClientMcpToolDefinition } from "./definitions.js";

export function validateClientMcpToolCall(
  tools: readonly ClientMcpToolDefinition[],
  toolName: string,
  input: unknown,
): string | null {
  if (typeof toolName !== "string" || !toolName.trim()) {
    return "Missing MCP tool name.";
  }
  const tool = tools.find((candidate) => candidate.name === toolName);
  if (!tool) {
    return `Unknown client MCP forwarding tool: ${toolName}`;
  }
  const args =
    input !== null && typeof input === "object" && !Array.isArray(input)
      ? (input as Record<string, unknown>)
      : {};
  return validateAgainstSchema(args, tool.inputSchema, toolName);
}

function validateAgainstSchema(
  value: unknown,
  schema: Record<string, unknown>,
  path: string,
): string | null {
  if (schema.type === "object") {
    if (typeof value !== "object" || value === null || Array.isArray(value)) {
      return `Invalid value for ${path}: expected object`;
    }
    const record = value as Record<string, unknown>;
    const required = schema.required;
    if (Array.isArray(required)) {
      for (const key of required) {
        if (typeof key !== "string") continue;
        if (!(key in record) || record[key] === undefined) {
          return `Invalid value for ${path}: missing required property ${key}`;
        }
      }
    }
    const properties = schema.properties;
    if (properties !== null && typeof properties === "object" && !Array.isArray(properties)) {
      const props = properties as Record<string, Record<string, unknown>>;
      for (const [key, propSchema] of Object.entries(props)) {
        if (!(key in record)) continue;
        const err = validateAgainstSchema(record[key], propSchema, `${path}.${key}`);
        if (err) return err;
      }
    }
    return null;
  }
  if (schema.type === "string" && typeof value !== "string") {
    return `Invalid value for ${path}: expected string`;
  }
  if (schema.type === "number" && typeof value !== "number") {
    return `Invalid value for ${path}: expected number`;
  }
  if (schema.type === "boolean" && typeof value !== "boolean") {
    return `Invalid value for ${path}: expected boolean`;
  }
  if (schema.type === "array" && !Array.isArray(value)) {
    return `Invalid value for ${path}: expected array`;
  }
  if (Array.isArray(schema.enum)) {
    if (!schema.enum.some((item) => Object.is(item, value))) {
      return `Invalid value for ${path}: not in enum`;
    }
  }
  return null;
}
