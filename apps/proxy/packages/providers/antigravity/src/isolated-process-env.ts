import path from "node:path";

const AGY_INTEROP_ENV_KEYS = [
  "PATH",
  "PATHEXT",
  "LANG",
  "LANGUAGE",
  "LC_ALL",
  "LC_CTYPE",
  "LC_MESSAGES",
  "TERM",
  "NO_COLOR",
  "COLORTERM",
  "DISPLAY",
  "WAYLAND_DISPLAY",
  "WSL_DISTRO_NAME",
  "WSL_INTEROP",
  "USER",
  "LOGNAME",
  "TMPDIR",
  "TEMP",
  "TMP",
] as const;

export function buildIsolatedAntigravityProcessEnv(
  storageRoot: string,
): Record<string, string> {
  const absoluteRoot = path.resolve(storageRoot);
  const env: Record<string, string> = {
    HOME: absoluteRoot,
    ANTIGRAVITY_APP_DATA_DIR: absoluteRoot,
    XDG_CONFIG_HOME: path.join(absoluteRoot, ".config"),
    XDG_DATA_HOME: path.join(absoluteRoot, ".local", "share"),
  };

  for (const key of AGY_INTEROP_ENV_KEYS) {
    const value = process.env[key];
    if (typeof value === "string" && value.length > 0) {
      env[key] = value;
    }
  }

  if (!env.DISPLAY) {
    env.DISPLAY = ":0";
  }

  for (const [key, value] of Object.entries(process.env)) {
    if (value !== undefined && key.toUpperCase().startsWith("LC_") && !(key in env)) {
      env[key] = value;
    }
  }

  return env;
}

export function isAntigravityStorageIsolation(
  explicitEnvironment?: Readonly<Record<string, string>>,
): boolean {
  if (!explicitEnvironment) {
    return false;
  }
  const home = explicitEnvironment.HOME;
  const appDataDir = explicitEnvironment.ANTIGRAVITY_APP_DATA_DIR;
  return (
    typeof home === "string" &&
    home.length > 0 &&
    typeof appDataDir === "string" &&
    home === appDataDir
  );
}