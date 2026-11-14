import { describe, expect, it } from "vitest";
import { AntigravityProvider } from "../src/provider.js";
import { executionInput, FakeAcpClient } from "./provider-test-helpers.js";

describe("AntigravityProvider tool_choice", () => {
  it("rejects tool_choice for non-existent function before execution", async () => {
    const client = new FakeAcpClient();
    const provider = new AntigravityProvider({ clientFactory: () => client });

    await expect(
      provider.execute(
        executionInput({
          tools: [{ name: "declared_tool", parameters: {} }],
          toolChoice: { type: "function", function: { name: "missing_tool" } },
        }),
      ),
    ).rejects.toThrow(
      "Requested tool 'missing_tool' in tool_choice was not found in declared tools.",
    );
  });

  it("rejects response when tool_choice='required' and no tool is invoked", async () => {
    const client = new FakeAcpClient();
    const provider = new AntigravityProvider({ clientFactory: () => client });

    const run = await provider.execute(
      executionInput({
        tools: [{ name: "my_tool", parameters: {} }],
        toolChoice: "required",
      }),
    );

    const result = await run.wait();
    expect(result.status).toBe("failed");
  });
});
