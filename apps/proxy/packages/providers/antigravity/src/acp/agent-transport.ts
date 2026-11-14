import type { ChildProcess, SpawnOptions } from "node:child_process";
import { spawnProcess, type SpawnFn } from "./spawn-types.js";
import type {
  AgentRequestListener,
  InvalidLineListener,
  JsonRpcMessage,
  JsonRpcRequest,
  JsonRpcResponse,
  NotificationListener,
} from "./types.js";
import {
  MessageStreamDecoder,
  buildNotification,
  buildRequest,
  serializeMessage,
} from "./json-rpc.js";
import {
  AcpNotConnectedError,
  AcpProcessError,
  AcpSpawnError,
  AcpTerminatedError,
  AcpTimeoutError,
} from "./errors.js";
import {
  hasChildExited,
  installAcpProcessExitHook,
  killChildTree,
  killRemainingProcessGroup,
  liveAcpChildren,
  waitForExitOrGrace,
} from "./agent-child-lifecycle.js";
import { mergeAgentSpawnEnvironment } from "./agent-process-env.js";

export type { SpawnFn };

export interface TransportOptions {
  readonly command: string;
  readonly args: readonly string[];
  readonly cwd?: string;
  readonly env?: Readonly<Record<string, string>>;
  readonly spawnFn?: SpawnFn;
  readonly exitListener?: (exitCode: number | null, signal: string | null) => void;
  readonly invalidLineListener?: InvalidLineListener;
  readonly agentRequestListener?: AgentRequestListener;
}

export class AgentProcessTransport {
  private child?: ChildProcess;
  private readonly notificationListeners = new Set<NotificationListener>();
  private readonly pending = new Map<
    number,
    {
      readonly resolve: (response: JsonRpcResponse) => void;
      readonly reject: (error: Error) => void;
      timer?: NodeJS.Timeout;
    }
  >();
  private terminated = false;

  constructor(
    private readonly options: TransportOptions,
    private readonly decoder: MessageStreamDecoder = new MessageStreamDecoder(
      options.invalidLineListener,
    ),
  ) {}

  public get isRunning(): boolean {
    const child = this.child;
    return (
      !this.terminated &&
      child !== undefined &&
      child.pid !== undefined &&
      child.exitCode === null &&
      child.signalCode === null
    );
  }

  public onNotification(listener: NotificationListener): () => void {
    this.notificationListeners.add(listener);
    return () => {
      this.notificationListeners.delete(listener);
    };
  }

  public start(): void {
    installAcpProcessExitHook();
    const spawnOptions: SpawnOptions = {
      cwd: this.options.cwd,
      env: mergeAgentSpawnEnvironment(process.env, this.options.env),
      stdio: ["pipe", "pipe", "pipe"] as SpawnOptions["stdio"],
      detached: process.platform !== "win32",
    };
    try {
      this.child = (this.options.spawnFn ?? spawnProcess)(
        this.options.command,
        [...this.options.args],
        spawnOptions,
      );
    } catch (error) {
      throw new AcpSpawnError(this.options.command, String(error));
    }
    const child = this.child;
    liveAcpChildren.add(child);
    child.on("error", (error: Error) => {
      this.failPending(`spawn failed: ${error.message}`, null, null);
    });
    child.on("exit", (code, signal) => {
      killRemainingProcessGroup(child);
      liveAcpChildren.delete(child);
      this.options.exitListener?.(code, signal);
      this.failPending(
        `process exited unexpectedly (code=${code ?? "null"}, signal=${signal ?? "null"})`,
        code,
        signal,
      );
    });
    child.on("close", () => {
      liveAcpChildren.delete(child);
    });

    const stdout = child.stdout;
    if (stdout) {
      stdout.setEncoding("utf8");
      stdout.on("data", (chunk: string) => {
        for (const message of this.decoder.push(chunk)) {
          this.dispatchMessage(message);
        }
      });
    }
    child.stderr?.on("data", () => {});
  }

  private dispatchMessage(message: JsonRpcMessage): void {
    const withMethod = message as JsonRpcRequest;
    if (typeof withMethod.method === "string") {
      this.handleAgentMessage(withMethod);
      return;
    }
    const response = message as JsonRpcResponse;
    const entry = this.pending.get(response.id);
    if (entry === undefined) {
      return;
    }
    this.pending.delete(response.id);
    if (entry.timer !== undefined) {
      clearTimeout(entry.timer);
    }
    entry.resolve(response);
  }

  private handleAgentMessage(request: JsonRpcRequest): void {
    if (request.id === undefined) {
      for (const listener of this.notificationListeners) {
        listener(request.method, request.params);
      }
      return;
    }
    const outcome = this.options.agentRequestListener?.(
      request.id,
      request.method,
      request.params,
    );
    if (outcome === undefined) {
      this.writeResponse({
        jsonrpc: "2.0",
        id: request.id,
        error: { code: -32601, message: "Method not supported by ACP client." },
      });
      return;
    }
    if ("error" in outcome) {
      this.writeResponse({ jsonrpc: "2.0", id: request.id, error: outcome.error });
      return;
    }
    this.writeResponse({ jsonrpc: "2.0", id: request.id, result: outcome.result });
  }

  private writeResponse(response: JsonRpcResponse): void {
    this.write(response);
  }

  public request(
    id: number,
    method: string,
    params: unknown,
    timeoutMs: number,
  ): Promise<JsonRpcResponse> {
    if (this.terminated) {
      return Promise.reject(new AcpTerminatedError("Transport has been terminated."));
    }
    if (!this.child || this.child.pid === undefined || this.child.exitCode !== null) {
      return Promise.reject(new AcpNotConnectedError("Agent process is not running."));
    }
    return new Promise<JsonRpcResponse>((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new AcpTimeoutError(method, timeoutMs));
      }, timeoutMs);
      this.pending.set(id, { resolve, reject, timer });
      this.write(buildRequest(id, method, params));
    });
  }

  public notify(method: string, params?: unknown): void {
    this.write(buildNotification(method, params));
  }

  public async terminate(graceMs = 3000): Promise<void> {
    this.terminated = true;
    const child = this.child;
    if (!child) {
      return;
    }
    if (hasChildExited(child)) {
      killRemainingProcessGroup(child);
      liveAcpChildren.delete(child);
      return;
    }
    let resolveExit: (() => void) | undefined;
    const exited = new Promise<void>((resolve) => {
      resolveExit = resolve;
      child.once("exit", resolve);
    });
    if (hasChildExited(child)) {
      resolveExit?.();
      return;
    }
    const pid = child.pid;
    if (pid !== undefined) {
      this.signalProcessGroup(pid, "SIGINT");
    } else {
      child.kill("SIGINT");
    }
    const graceful = await waitForExitOrGrace(exited, graceMs);
    if (!graceful && !hasChildExited(child)) {
      killChildTree(child);
      if (!hasChildExited(child)) {
        await exited;
      }
    }
  }

  private signalProcessGroup(pid: number, signal: NodeJS.Signals): void {
    try {
      process.kill(-pid, signal);
    } catch {
      try {
        this.child?.kill(signal);
      } catch {
        // already dead
      }
    }
  }

  private write(message: JsonRpcMessage): void {
    const stdin = this.child?.stdin;
    if (!stdin?.writable) {
      throw new AcpNotConnectedError("Agent stdin is unavailable.");
    }
    stdin.write(serializeMessage(message));
  }

  private failPending(
    reason: string,
    exitCode: number | null,
    signal: string | null,
  ): void {
    const entries = [...this.pending.values()];
    this.pending.clear();
    for (const entry of entries) {
      if (entry.timer !== undefined) {
        clearTimeout(entry.timer);
      }
      entry.reject(new AcpProcessError(`ACP agent ${reason}.`, exitCode, signal));
    }
  }
}