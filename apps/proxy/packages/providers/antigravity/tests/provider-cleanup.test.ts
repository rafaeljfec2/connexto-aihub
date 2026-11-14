import { InMemorySessionStore } from "@agent-runtime-gateway/core";
import { describe, expect, it } from "vitest";
import type { AcpSessionRecord } from "../src/session-store.js";
import { AntigravityProvider } from "../src/provider.js";
import { FakeAcpClient } from "./provider-test-helpers.js";

describe("AntigravityProvider session cleanup", () => {
  it("closePersistedAcpSessions closes mapped ids and ids from listSessions", async () => {
    class ListCloseFake extends FakeAcpClient {
      public override listSessions(): Promise<{
        sessions: readonly { sessionId: string }[];
      }> {
        this.calls.push("list");
        return Promise.resolve({ sessions: [{ sessionId: "acp-from-list" }] });
      }
    }

    const mem = new InMemorySessionStore<AcpSessionRecord>();
    await mem.set("antigravity", "gw-x", {
      acpSessionId: "acp-mapped",
      touchedAt: Date.now(),
    });

    const provider = new AntigravityProvider({
      clientFactory: () => new ListCloseFake(),
      sessionStore: mem,
      defaultWorkingDirectory: "/tmp/ag-clear-test",
    });

    const report = await provider.closePersistedAcpSessions();

    expect(report.mapped_acp_ids_count).toBe(1);
    expect(report.listed_acp_sessions_count).toBe(1);
    expect(report.acp_sessions_closed).toBe(2);
    expect(report.acp_close_failures).toBe(0);
  });
});
