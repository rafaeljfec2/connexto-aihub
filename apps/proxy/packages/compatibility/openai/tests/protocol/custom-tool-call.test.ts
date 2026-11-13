import { describe, expect, it } from "vitest";
import { responseObject } from "../../src/protocol/formatters.js";
import { responseToolCallEvents } from "../../src/protocol/sse.js";
import { responseInputToTextAndImages } from "../../src/protocol/transcripts/builder.js";

describe("Codex grammar tool calls", () => {
  it("emits a custom tool call whose input is raw JavaScript", () => {
    const toolCall = {
      id: "call_exec_1",
      type: "custom" as const,
      function: {
        name: "exec",
        arguments: 'const result = await tools.exec_command({"cmd":"git status -sb"});\ntext(result);',
      },
    };
    const frames = Buffer.concat(
      responseToolCallEvents({ id: "resp_test", toolCall, outputIndex: 1 }),
    ).toString("utf8");

    expect(frames).toContain("response.custom_tool_call_input.done");
    expect(frames).toContain('"type":"custom_tool_call"');
    expect(frames).not.toContain("response.function_call_arguments.done");

    const completed = responseObject({
      id: "resp_test",
      created: 1,
      model: "composer-2.5",
      text: "",
      toolCalls: [toolCall],
      promptChars: 1,
    });
    const output = completed.output as Array<Record<string, unknown>>;
    expect(output.some((item) => item.type === "custom_tool_call" && item.input === toolCall.function.arguments)).toBe(true);
  });

  it("feeds custom tool output back as a local tool result", () => {
    const { text } = responseInputToTextAndImages([
      { type: "custom_tool_call", call_id: "call_exec_1", name: "exec", input: "text(1);" },
      { type: "custom_tool_call_output", call_id: "call_exec_1", output: "TOKEN-OK" },
    ]);

    expect(text).toContain("LOCAL TOOL RESULT:");
    expect(text).toContain("TOKEN-OK");
  });
});
