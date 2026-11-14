import type { ContentBlock } from "@agent-runtime-gateway/provider-antigravity/acp";
import { describe, expect, it } from "vitest";
import { AntigravityProvider } from "../src/provider.js";
import { executionInput, FakeAcpClient } from "./provider-test-helpers.js";

describe("AntigravityProvider prompt blocks", () => {
  it("appends image content blocks after the text block", async () => {
    const client = new FakeAcpClient();
    const provider = new AntigravityProvider({ clientFactory: () => client });

    await provider.execute(
      executionInput({
        message: "Describe these images",
        images: [
          { mimeType: "image/png", data: "AAAAAFBO" },
          { mimeType: "image/jpeg", data: "AAABAAA" },
        ],
      }),
    );

    expect(client.lastBlocks).toEqual([
      { type: "text", text: "Describe these images" },
      { type: "image", mimeType: "image/png", data: "AAAAAFBO" },
      { type: "image", mimeType: "image/jpeg", data: "AAABAAA" },
    ]);
  });

  it("sends a text-only block list when no images are provided", async () => {
    const client = new FakeAcpClient();
    const provider = new AntigravityProvider({ clientFactory: () => client });

    await provider.execute(executionInput());

    expect(client.lastBlocks).toEqual([{ type: "text", text: "Hello" }]);
  });

  it("appends anti-parallel instruction when parallelToolCalls is false", async () => {
    const client = new FakeAcpClient();
    const provider = new AntigravityProvider({ clientFactory: () => client });

    const run = await provider.execute(
      executionInput({
        tools: [{ name: "my_tool", parameters: {} }],
        parallelToolCalls: false,
      }),
    );
    await run.wait();

    const textBlocks = (client.lastBlocks ?? []).filter(
      (block): block is Extract<ContentBlock, { type: "text" }> => block.type === "text",
    );
    expect(textBlocks[0]?.text).toContain("Parallel tool calling is disabled");
  });
});
