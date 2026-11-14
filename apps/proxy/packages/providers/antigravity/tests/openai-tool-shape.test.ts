import { describe, expect, it } from "vitest";
import {
  delegatedToolCallPayload,
  mcpInputSchemaForTool,
} from "../src/mcp/openai-tool-shape.js";

describe("openai tool shape", () => {
  it("keeps function tools as JSON arguments", () => {
    const payload = delegatedToolCallPayload(
      { name: "get_stream_telemetry", parameters: { type: "object" } },
      { channel: "alpha-live" },
    );
    expect(payload).toEqual({ arguments: JSON.stringify({ channel: "alpha-live" }) });
  });

  it("emits raw input for grammar tools as a custom tool call", () => {
    const spec = { name: "exec", callKind: "custom" as const, inputFormat: "grammar" as const };
    expect(mcpInputSchemaForTool(spec).required).toEqual(["input"]);
    const input = mcpInputSchemaForTool(spec).properties as { input: { description: string } };
    expect(input.input.description).toContain("tools.exec_command");
    expect(input.input.description).toContain("run_command is not a tool");
    expect(
      delegatedToolCallPayload(spec, { input: "await tools.exec_command({cmd:\"git status\"})" }),
    ).toEqual({
      arguments: "await tools.exec_command({cmd:\"git status\"})",
      callKind: "custom",
    });
  });
});
