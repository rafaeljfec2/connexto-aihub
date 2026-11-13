import { describe, expect, it } from "vitest";
import { prepareResponsesPassthrough } from "../../src/protocol/prepare.js";
import {
  collectResponsesToolSpecs,
  flattenAdditionalToolsFromInput,
} from "../../src/protocol/tools/responses-tools.js";

describe("Codex additional_tools inventory", () => {
  const codexShapedBody = {
    model: "aihub/auto",
    tool_choice: "auto",
    stream: false,
    input: [
      {
        type: "additional_tools",
        tools: [
          {
            type: "namespace",
            name: "functions",
            tools: [
              { type: "custom", name: "exec", description: "run js" },
              {
                type: "function",
                name: "wait",
                parameters: { type: "object", properties: {} },
              },
            ],
          },
          {
            type: "namespace",
            name: "clock",
            tools: [
              {
                type: "function",
                name: "sleep",
                parameters: {
                  type: "object",
                  properties: { duration_ms: { type: "number" } },
                  required: ["duration_ms"],
                },
              },
            ],
          },
        ],
      },
      {
        type: "message",
        role: "user",
        content: [{ type: "input_text", text: "try again" }],
      },
    ],
  };

  it("flattens nested namespaces from input additional_tools", () => {
    const tools = flattenAdditionalToolsFromInput(codexShapedBody.input);
    expect(tools.map((tool) => tool.name).sort()).toEqual(["exec", "sleep", "wait"]);
  });

  it("collects tools when top-level tools is absent", () => {
    const tools = collectResponsesToolSpecs(codexShapedBody);
    expect(tools).toHaveLength(3);
    expect(tools.some((tool) => tool.name === "exec")).toBe(true);
  });

  it("prepareResponsesPassthrough marks agent mode and exposes tools", () => {
    const prepared = prepareResponsesPassthrough(codexShapedBody, {
      id: "composer-2.5",
    });
    expect(prepared.prompt.mode).toBe("agent");
    expect(prepared.tools.map((tool) => tool.name).sort()).toEqual([
      "exec",
      "sleep",
      "wait",
    ]);
  });
});
