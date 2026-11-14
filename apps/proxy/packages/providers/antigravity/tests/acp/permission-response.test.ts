import { describe, expect, it } from "vitest";
import { permissionResponse } from "../../src/acp/permission-response.js";

describe("permissionResponse", () => {
  it("selects allow_always so the MCP tool is not recorded as a user denial", () => {
    expect(
      permissionResponse({
        options: [
          { optionId: "allow-always", kind: "allow_always" },
          { optionId: "reject-once", kind: "reject_once" },
        ],
      }),
    ).toEqual({ result: { outcome: { outcome: "selected", optionId: "allow-always" } } });
  });

  it("falls back to allow_once when always is absent", () => {
    expect(
      permissionResponse({
        options: [{ optionId: "allow-once", kind: "allow_once" }],
      }),
    ).toEqual({ result: { outcome: { outcome: "selected", optionId: "allow-once" } } });
  });

  it("cancels when the agent offers no allow option", () => {
    expect(permissionResponse({ options: [{ optionId: "reject-once", kind: "reject_once" }] })).toEqual({
      result: { outcome: { outcome: "cancelled" } },
    });
  });
});
