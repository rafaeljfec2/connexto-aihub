import crypto from "node:crypto";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { credentialFilePaths, detectAccountFromHomeDir } from "./auth-credential-store.js";
import { processAuthCallback } from "./auth-callback-flow.js";
import {
  ANTIGRAVITY_VIABILITY_LIMITATIONS,
  ANTIGRAVITY_VIABILITY_GATE,
  GOOGLE_OAUTH_REDIRECT_URI,
  buildGoogleOAuthAuthorizationUrl,
  defaultGoogleTokenExchanger,
} from "./google-oauth-config.js";
import type {
  AntigravityAccountInfo,
  AntigravityAuthCallbackResult,
  AntigravityAuthManagerOptions,
  AntigravityAuthOperation,
  AntigravityAuthOperationSummary,
  AntigravityAuthStartResponse,
  AntigravityAuthStatusResponse,
  AntigravityTokens,
} from "./types.js";

export { defaultGoogleTokenExchanger } from "./google-oauth-config.js";

const DEFAULT_OPERATION_TTL_MS = 600_000; // 10 minutes

export class AntigravityAuthManager {
  private readonly homeDir: string;
  private readonly operations = new Map<string, AntigravityAuthOperation>();
  private readonly stateToOperationId = new Map<string, string>();
  private readonly operationTtlMs: number;
  private readonly defaultRedirectUri?: string;
  private readonly tokenExchanger?: AntigravityAuthManagerOptions["tokenExchanger"];
  private readonly hasExplicitTokenExchanger: boolean;
  private readonly onTokensPersisted?: (
    tokens: AntigravityTokens,
    email?: string,
  ) => Promise<void>;
  private readonly accountDetector: () => Promise<AntigravityAccountInfo>;
  private readonly cliLoginSpawner?: AntigravityAuthManagerOptions["cliLoginSpawner"];
  private lastActiveOperationId?: string;

  constructor(options: AntigravityAuthManagerOptions = {}) {
    const defaultStorage =
      process.env.ANTIGRAVITY_STORAGE_PATH ??
      path.join(os.homedir(), ".connexto-proxy", "antigravity");
    this.homeDir = options.homeDir ?? defaultStorage;
    this.operationTtlMs = options.operationTtlMs ?? DEFAULT_OPERATION_TTL_MS;
    this.accountDetector =
      options.accountDetector ?? (() => detectAccountFromHomeDir(this.homeDir));
    this.cliLoginSpawner = options.cliLoginSpawner;
    this.defaultRedirectUri = options.defaultRedirectUri;
    this.hasExplicitTokenExchanger = options.tokenExchanger !== undefined;
    this.tokenExchanger = options.tokenExchanger ?? defaultGoogleTokenExchanger;
    this.onTokensPersisted = options.onTokensPersisted;
  }

  public async startAuth(params?: {
    redirectUri?: string;
  }): Promise<AntigravityAuthStartResponse> {
    this.purgeExpiredOperations();

    const operationId = crypto.randomUUID();
    const state = crypto.randomBytes(24).toString("base64url");
    const codeVerifier = crypto.randomBytes(32).toString("base64url");
    const codeChallenge = crypto
      .createHash("sha256")
      .update(codeVerifier)
      .digest("base64url");

    const redirectUri =
      params?.redirectUri ?? this.defaultRedirectUri ?? GOOGLE_OAUTH_REDIRECT_URI;

    const authUrl = buildGoogleOAuthAuthorizationUrl({
      redirectUri,
      state,
      codeChallenge,
    });
    const now = new Date();
    const expiresAt = new Date(now.getTime() + this.operationTtlMs).toISOString();

    const operation: AntigravityAuthOperation = {
      operationId,
      status: "pending",
      authUrl,
      state,
      codeVerifier,
      redirectUri,
      createdAt: now.toISOString(),
      expiresAt,
    };

    this.operations.set(operationId, operation);
    this.stateToOperationId.set(state, operationId);
    this.lastActiveOperationId = operationId;

    if (this.cliLoginSpawner) {
      await this.cliLoginSpawner({ operationId, authUrl, state });
    }

    return {
      operationId,
      status: "pending",
      authUrl,
      state,
      expiresAt,
    };
  }

  public async handleCallback(params: {
    state?: string;
    code?: string;
    error?: string;
    error_description?: string;
  }): Promise<AntigravityAuthCallbackResult> {
    const tokenExchanger = this.tokenExchanger ?? defaultGoogleTokenExchanger;
    return processAuthCallback({
      homeDir: this.homeDir,
      operations: this.operations,
      stateToOperationId: this.stateToOperationId,
      tokenExchanger,
      hasExplicitTokenExchanger: this.hasExplicitTokenExchanger,
      onTokensPersisted: this.onTokensPersisted,
      accountDetector: this.accountDetector,
      callback: params,
    });
  }

  public getOperation(
    operationId: string,
  ): AntigravityAuthOperationSummary | undefined {
    const operation = this.operations.get(operationId);
    if (!operation) return undefined;

    if (
      operation.status === "pending" &&
      Date.now() > new Date(operation.expiresAt).getTime()
    ) {
      const expiredOp: AntigravityAuthOperation = { ...operation, status: "expired" };
      this.operations.set(operationId, expiredOp);
      return this.toSummary(expiredOp);
    }

    return this.toSummary(operation);
  }

  public async getStatus(): Promise<AntigravityAuthStatusResponse> {
    this.purgeExpiredOperations();
    const accountInfo = await this.accountDetector().catch(
      (): AntigravityAccountInfo => ({ authenticated: false }),
    );

    const activeOperation = this.lastActiveOperationId
      ? this.getOperation(this.lastActiveOperationId)
      : undefined;

    return {
      authenticated: accountInfo.authenticated,
      account: accountInfo.account,
      lastCheckedAt: new Date().toISOString(),
      viabilityGate: ANTIGRAVITY_VIABILITY_GATE,
      limitations: ANTIGRAVITY_VIABILITY_LIMITATIONS,
      activeOperation,
    };
  }

  public resetForTest(): void {
    this.operations.clear();
    this.stateToOperationId.clear();
    this.lastActiveOperationId = undefined;
  }

  public async clearCredentials(): Promise<{
    removedFiles: string[];
    message: string;
  }> {
    this.operations.clear();
    this.stateToOperationId.clear();
    this.lastActiveOperationId = undefined;

    const removedFiles: string[] = [];
    for (const filePath of credentialFilePaths(this.homeDir)) {
      try {
        await fs.unlink(filePath);
        removedFiles.push(filePath);
      } catch (err: unknown) {
        if ((err as NodeJS.ErrnoException).code !== "ENOENT") {
          // File might not exist or failed to remove; do not crash
        }
      }
    }

    return {
      removedFiles,
      message: "Antigravity credentials and active sessions purged successfully.",
    };
  }

  private toSummary(op: AntigravityAuthOperation): AntigravityAuthOperationSummary {
    return {
      operationId: op.operationId,
      status: op.status,
      createdAt: op.createdAt,
      expiresAt: op.expiresAt,
      error: op.error,
      account: op.account,
    };
  }

  private purgeExpiredOperations(): void {
    const now = Date.now();
    for (const [id, op] of this.operations.entries()) {
      if (op.status === "pending" && now > new Date(op.expiresAt).getTime()) {
        this.operations.set(id, { ...op, status: "expired" });
      }
    }
  }
}