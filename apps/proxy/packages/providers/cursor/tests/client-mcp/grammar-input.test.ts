import { describe, expect, it } from "vitest";
import { grammarSourceFromArguments } from "../../src/client-mcp/grammar-input.js";

describe("grammarSourceFromArguments", () => {
  it("forwards the custom tool input string unchanged", () => {
    const source = grammarSourceFromArguments({
      input: 'const result = await tools.exec_command({cmd:"git status -sb"});\ntext(result);',
    });

    expect(source).toBe(
      'const result = await tools.exec_command({cmd:"git status -sb"});\ntext(result);',
    );
  });

  it("unwraps a JSON string stored in args before reading input", () => {
    const source = grammarSourceFromArguments({
      args: JSON.stringify({ input: "print(1)" }),
    });

    expect(source).toBe("print(1)");
  });
});
