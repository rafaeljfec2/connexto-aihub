import { randomUUID } from "node:crypto";
import type {
  AgentExecutionInput,
  AgentProvider,
  AgentRun,
  Model,
  NamespacedSessionStore,
  SessionStore,
} from "@agent-runtime-gateway/core";
import { AcpTurnInProgressError } from "./acp/index.js";
import {
  createDefaultClient,
  terminatePreservingCause,
  type AntigravityAcpClient,
} from "./acp-client.js";
import {
  closeAntigravityAcpSessionsWithReport,
  resolveWorkingDirectory,
  type AntigravityRuntimeSessionClearReport,
} from "./acp-session-cleanup.js";
import { AsyncEventQueue } from "./async-event-queue.js";
import { createDelegatedTurnController } from "./delegated-tool-turn.js";
import { acpModelId, ANTIGRAVITY_MODELS } from "./models.js";
import { PendingToolRegistry } from "./pending-tools.js";
import {
  AntigravitySessionStore,
  type AcpSessionRecord,
  type AntigravitySessionStoreOptions,
} from "./session-store.js";
import { createAntigravityRun } from "./turn-runner.js";
import { assertRequestedToolIsDeclared } from "./turn-disposition.js";

export type { AntigravityAcpClient } from "./acp-client.js";
export type { AntigravityRuntimeSessionClearReport } from "./acp-session-cleanup.js";
export { acpTurnDisposition } from "./turn-disposition.js";

export interface AntigravityProviderOptions {
  readonly clientFactory?: (cwd: string) => AntigravityAcpClient;
  readonly defaultWorkingDirectory?: string;
  readonly homeDir?: string;
  readonly cancelGraceMs?: number;
  readonly sessionOptions?: AntigravitySessionStoreOptions;
  readonly sessionStore?:
    | NamespacedSessionStore<AcpSessionRecord>
    | SessionStore<AcpSessionRecord>;
  readonly pendingTools?: PendingToolRegistry;
}

const DEFAULT_CANCEL_GRACE_MS = 3_000;
const LOCAL_CREDENTIAL_TYPE = "local";

export class AntigravityProvider implements AgentProvider {
  public readonly id = "antigravity";
  public readonly pendingTools: PendingToolRegistry;

  private readonly clientFactory: (cwd: string) => AntigravityAcpClient;
  private readonly defaultWorkingDirectory: string;
  private readonly homeDir?: string;
  private readonly cancelGraceMs: number;
  private readonly sessions: AntigravitySessionStore;
  private readonly activeSessions = new Set<string>();

  constructor(options: AntigravityProviderOptions = {}) {
    this.homeDir = options.homeDir;
    this.clientFactory =
      options.clientFactory ?? ((cwd: string) => createDefaultClient(cwd, this.homeDir));
    this.defaultWorkingDirectory = options.defaultWorkingDirectory ?? process.cwd();
    this.cancelGraceMs = options.cancelGraceMs ?? DEFAULT_CANCEL_GRACE_MS;
    this.sessions = new AntigravitySessionStore({
      ...options.sessionOptions,
      store: options.sessionOptions?.store ?? options.sessionStore,
    });
    this.pendingTools = options.pendingTools ?? new PendingToolRegistry();
  }

  public listModels(): Promise<Model[]> {
    return Promise.resolve(ANTIGRAVITY_MODELS.map((model) => ({ ...model })));
  }

  /**
   * Step 1 of runtime clear: close ACP sessions upstream — first by persisted
   * gateway→ACP map, then by `session/list` and `session/close` for anything
   * still live in the local runtime. Returns a verifiable report.
   */
  public async closePersistedAcpSessions(): Promise<AntigravityRuntimeSessionClearReport> {
    let mappedIds: string[];
    try {
      mappedIds = await this.sessions.listAllAcpSessionIds();
    } catch {
      mappedIds = [];
    }
    return closeAntigravityAcpSessionsWithReport(
      mappedIds,
      this.defaultWorkingDirectory,
      this.clientFactory,
    );
  }

  /**
   * Step 2 of runtime clear: drop in-memory gateway→ACP maps and execution locks
   * after `sessionStore.clear("antigravity")` removed JSON on disk.
   */
  public clearSessionCaches(): void {
    this.sessions.clearInMemory();
    this.activeSessions.clear();
    this.pendingTools.clear();
  }

  public async execute(input: AgentExecutionInput): Promise<AgentRun> {
    const acpModel = this.assertReady(input);
    const cwd = resolveWorkingDirectory(input.workingDirectory, this.defaultWorkingDirectory);
    const client = this.clientFactory(cwd);
    this.activeSessions.add(input.sessionId);
    const runId = `run_${randomUUID().replaceAll("-", "")}`;
    const queue = new AsyncEventQueue();
    const delegation = createDelegatedTurnController({
      execution: input,
      runId,
      queue,
      pendingTools: this.pendingTools,
    });
    try {
      const started = await delegation.start();
      const acpSessionId = await this.openAcpSession({
        client,
        execution: input,
        cwd,
        acpModel,
        mcpServers: started.mcpServers,
      });
      input.signal?.throwIfAborted();
      return createAntigravityRun({
        client,
        input,
        acpSessionId,
        runId,
        queue,
        bridge: started.bridge,
        recordedToolCalls: delegation.recordedToolCalls,
        delegatedTurn: delegation.delegatedTurn,
        cancelGraceMs: this.cancelGraceMs,
        onSessionReleased: () => {
          this.activeSessions.delete(input.sessionId);
        },
      });
    } catch (error) {
      return this.abortSetup(client, input.sessionId, delegation.bridge, error);
    }
  }

  private assertReady(input: AgentExecutionInput): string {
    const acpModel = acpModelId(input.model);
    if (acpModel === undefined) {
      throw new Error("Antigravity provider received an unsupported model.");
    }
    if (input.credential.type !== LOCAL_CREDENTIAL_TYPE) {
      throw new Error("Antigravity provider requires local ACP authentication.");
    }
    input.signal?.throwIfAborted();
    if (this.activeSessions.has(input.sessionId)) throw new AcpTurnInProgressError();
    assertRequestedToolIsDeclared(input.toolChoice, input.tools);
    return acpModel;
  }

  private async openAcpSession(session: {
    readonly client: AntigravityAcpClient;
    readonly execution: AgentExecutionInput;
    readonly cwd: string;
    readonly acpModel: string;
    readonly mcpServers: unknown[] | undefined;
  }): Promise<string> {
    await session.client.connect();
    if (typeof session.client.authenticate === "function") {
      await session.client.authenticate("oauth-personal");
    }
    const acpSessionId = await this.acquireAcpSession(session);
    await session.client.setSessionConfig(acpSessionId, "model", session.acpModel);
    if (session.execution.reasoningEffort && session.acpModel === "gemini-3.8-flash") {
      await session.client.setSessionConfig(
        acpSessionId,
        "thought_level",
        session.execution.reasoningEffort,
      );
    }
    return acpSessionId;
  }

  private async acquireAcpSession(session: {
    readonly client: AntigravityAcpClient;
    readonly execution: AgentExecutionInput;
    readonly cwd: string;
    readonly mcpServers: unknown[] | undefined;
  }): Promise<string> {
    const gatewaySessionId = session.execution.sessionId;
    const savedSessionId = await this.sessions.resolve(gatewaySessionId);
    if (savedSessionId !== undefined && (await this.tryResume(session, savedSessionId))) {
      return savedSessionId;
    }
    const created = await session.client.createSession(session.cwd, session.mcpServers);
    await this.sessions.setPersisted(gatewaySessionId, created.sessionId);
    return created.sessionId;
  }

  private async tryResume(
    session: {
      readonly client: AntigravityAcpClient;
      readonly execution: AgentExecutionInput;
      readonly cwd: string;
      readonly mcpServers: unknown[] | undefined;
    },
    savedSessionId: string,
  ): Promise<boolean> {
    try {
      await session.client.resumeSession(savedSessionId, session.cwd, session.mcpServers);
      return true;
    } catch {
      await this.sessions.delete(session.execution.sessionId);
      return false;
    }
  }

  private async abortSetup(
    client: AntigravityAcpClient,
    sessionId: string,
    bridge: { dispose(): Promise<void> } | undefined,
    error: unknown,
  ): Promise<never> {
    if (bridge) await bridge.dispose().catch(() => {});
    this.activeSessions.delete(sessionId);
    await terminatePreservingCause(client, error);
    throw error;
  }
}
