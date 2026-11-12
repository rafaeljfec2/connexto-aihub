import { collectUsageViaBridge } from "./cursor-usage-bridge-collect.js";
import { collectUsageLocally } from "./cursor-usage-local-collect.js";
import {
  cancelCursorUsageAuth,
  getCursorCollectSessionStatus,
  getCursorInteractionPage,
  getCursorUsageAuthStatus,
  startCursorUsageAuth,
} from "./cursor-usage-auth.js";
import {
  createCursorUsageRuntime,
  type CursorUsageRuntime,
} from "./cursor-usage-runtime.js";
import type {
  CursorUsageAuthStartResult,
  CursorUsageAuthStatusResult,
  CursorUsageCollectResult,
  CursorUsageCollectSessionStatus,
  CursorUsageServiceOptions,
} from "./types.js";

export class CursorUsageService {
  private readonly runtime: CursorUsageRuntime;

  constructor(options: CursorUsageServiceOptions = {}) {
    this.runtime = createCursorUsageRuntime(options);
  }

  public async collectUsage(
    accountId: string,
  ): Promise<CursorUsageCollectResult> {
    return this.runtime.locks.run(accountId, async () => {
      if (this.runtime.hostBridgeClient) {
        return collectUsageViaBridge(this.runtime, accountId);
      }
      return collectUsageLocally(this.runtime, accountId);
    });
  }

  public async startAuth(
    accountId: string,
  ): Promise<CursorUsageAuthStartResult> {
    return startCursorUsageAuth(this.runtime, accountId);
  }

  public async getAuthStatus(
    sessionId: string,
  ): Promise<CursorUsageAuthStatusResult> {
    return getCursorUsageAuthStatus(this.runtime, sessionId);
  }

  public async getCollectSessionStatus(
    accountId: string,
  ): Promise<CursorUsageCollectSessionStatus> {
    return getCursorCollectSessionStatus(this.runtime, accountId);
  }

  public getInteractionPage(sessionId: string): { url: string } | undefined {
    return getCursorInteractionPage(this.runtime, sessionId);
  }

  public async cancelAuth(sessionId: string): Promise<void> {
    await cancelCursorUsageAuth(this.runtime, sessionId);
  }

  public async shutdown(): Promise<void> {
    await this.runtime.browserPool.shutdown();
    for (const context of this.runtime.liveContexts.values()) {
      await context.close();
    }
    this.runtime.liveContexts.clear();
  }

  public async clearPersistedSessions(): Promise<void> {
    this.runtime.authOperations.clear();
    this.runtime.sessionStore.clearMemory();
    await this.runtime.browserPool.shutdown();
    if (!this.runtime.hostBridgeClient) {
      return;
    }
    try {
      const sessions = await this.runtime.hostBridgeClient.listSessions();
      for (const session of sessions) {
        if (session.target === "cursor" || session.principalId === "cursor") {
          await this.runtime.hostBridgeClient
            .closeSession(session.sessionId, session.principalId)
            .catch(() => undefined);
        }
      }
    } catch {
      // Best-effort remote bridge cleanup
    }
  }
}
