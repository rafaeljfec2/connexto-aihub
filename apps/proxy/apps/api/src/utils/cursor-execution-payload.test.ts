import { describe, expect, it } from "vitest";
import type { Request } from "express";
import { buildCursorExecutionContext } from "./cursor-execution-payload.js";
import type { PreparedRequest } from "@agent-runtime-gateway/compatibility-openai";

function hubRequest(): Request {
  return {
    get(name: string) {
      return name.toLowerCase() === "x-gateway-api-key" ? "test-gw" : undefined;
    },
  } as Request;
}

describe("buildCursorExecutionContext", () => {
  it("maps Codex additional_tools from passthrough body into clientTools", () => {
    const prepared = {
      model: "composer-2.5",
      prompt: { text: "hi", mode: "agent" as const },
      stream: false,
      includeUsage: false,
      promptChars: 2,
      responseMetadata: {},
      tools: [],
      requiresLocalTool: false,
      storeResponse: false,
      passthroughBody: {
        model: "composer-2.5",
        tool_choice: "auto",
        input: [
          {
            type: "additional_tools",
            tools: [
              {
                type: "namespace",
                name: "functions",
                tools: [
                  {
                    type: "custom",
                    name: "exec",
                    description: "run",
                    format: { type: "grammar", syntax: "lark", definition: "start: SOURCE" },
                  },
                ],
              },
            ],
          },
        ],
      },
    } as PreparedRequest;

    const ctx = buildCursorExecutionContext(hubRequest(), prepared);
    expect(ctx?.mode).toBe("edge_owned");
    expect(ctx?.clientTools).toEqual([
      expect.objectContaining({ name: "exec", inputFormat: "grammar" }),
    ]);
  });
});
