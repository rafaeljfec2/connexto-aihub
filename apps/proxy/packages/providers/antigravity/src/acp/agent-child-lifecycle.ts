import type { ChildProcess } from "node:child_process";

/** Children alive under ACP transport; killed (group-level) on process exit. */
export const liveAcpChildren = new Set<ChildProcess>();
let exitHookInstalled = false;

export function installAcpProcessExitHook(): void {
  if (exitHookInstalled === true) {
    return;
  }
  exitHookInstalled = true;
  process.on("exit", () => {
    for (const child of liveAcpChildren) {
      if (child.exitCode !== null && child.exitCode !== undefined) {
        continue;
      }
      killChildTree(child);
    }
    liveAcpChildren.clear();
  });
}

export function killChildTree(child: ChildProcess): void {
  const pid = child.pid;
  if (pid === undefined) {
    try {
      child.kill("SIGKILL");
    } catch {
      // process already gone
    }
    return;
  }
  try {
    process.kill(-pid, "SIGKILL");
  } catch {
    try {
      child.kill("SIGKILL");
    } catch {
      // process already gone
    }
  }
}

export function killRemainingProcessGroup(child: ChildProcess): void {
  if (process.platform === "win32" || child.pid === undefined) {
    return;
  }
  try {
    process.kill(-child.pid, "SIGKILL");
  } catch {
    // Process group has already exited.
  }
}

export function hasChildExited(child: ChildProcess): boolean {
  return child.exitCode !== null || child.signalCode !== null;
}

export async function waitForExitOrGrace(
  exited: Promise<void>,
  graceMs: number,
): Promise<boolean> {
  let timer: NodeJS.Timeout | undefined;
  const grace = new Promise<boolean>((resolve) => {
    timer = setTimeout(() => resolve(false), graceMs);
  });
  const finished = await Promise.race([exited.then(() => true), grace]);
  if (timer !== undefined) {
    clearTimeout(timer);
  }
  return finished;
}