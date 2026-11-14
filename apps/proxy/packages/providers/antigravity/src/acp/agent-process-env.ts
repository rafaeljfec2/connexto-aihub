import { isAntigravityStorageIsolation } from "../isolated-process-env.js";

const INHERITED_ENV_KEYS = new Set([
  "APPDATA",
  "COLORTERM",
  "COMSPEC",
  "HOME",
  "HOMEDRIVE",
  "HOMEPATH",
  "LANG",
  "LANGUAGE",
  "LOCALAPPDATA",
  "LOGNAME",
  "NO_COLOR",
  "PATH",
  "PATHEXT",
  "SYSTEMROOT",
  "TEMP",
  "TERM",
  "TMP",
  "TMPDIR",
  "USER",
  "USERPROFILE",
  "WINDIR",
  "DBUS_SESSION_BUS_ADDRESS",
  "DISPLAY",
  "WAYLAND_DISPLAY",
  "WSL_DISTRO_NAME",
  "WSL_INTEROP",
  "XDG_RUNTIME_DIR",
  "XDG_CACHE_HOME",
  "XDG_CONFIG_HOME",
  "XDG_DATA_HOME",
  "XDG_STATE_HOME",
]);

const ISOLATION_BLOCKED_HOST_ENV_KEYS = new Set([
  "DBUS_SESSION_BUS_ADDRESS",
  "XDG_CACHE_HOME",
  "XDG_CONFIG_HOME",
  "XDG_DATA_HOME",
  "XDG_RUNTIME_DIR",
  "XDG_STATE_HOME",
]);

export function mergeAgentSpawnEnvironment(
  hostEnvironment: NodeJS.ProcessEnv,
  explicitEnvironment?: Readonly<Record<string, string>>,
): NodeJS.ProcessEnv {
  const storageIsolated = isAntigravityStorageIsolation(explicitEnvironment);
  const safeEnvironment: NodeJS.ProcessEnv = {};
  for (const [key, value] of Object.entries(hostEnvironment)) {
    const normalizedKey = key.toUpperCase();
    if (storageIsolated && ISOLATION_BLOCKED_HOST_ENV_KEYS.has(normalizedKey)) {
      continue;
    }
    if (
      value !== undefined &&
      (INHERITED_ENV_KEYS.has(normalizedKey) || normalizedKey.startsWith("LC_"))
    ) {
      safeEnvironment[key] = value;
    }
  }
  const merged: NodeJS.ProcessEnv = { ...safeEnvironment, ...explicitEnvironment };
  if (storageIsolated) {
    delete merged.DBUS_SESSION_BUS_ADDRESS;
  }
  return merged;
}