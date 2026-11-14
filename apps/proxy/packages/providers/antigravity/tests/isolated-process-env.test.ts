import os from "node:os";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { mergeAgentSpawnEnvironment } from "../src/acp/agent-process-env.js";
import {
  buildIsolatedAntigravityProcessEnv,
  isAntigravityStorageIsolation,
} from "../src/isolated-process-env.js";

describe("isolated Antigravity process environment", () => {
  it("builds storage-root HOME, XDG paths, and default DISPLAY", () => {
    const storage = path.join(os.tmpdir(), "ag-isolated-env-test");
    const env = buildIsolatedAntigravityProcessEnv(storage);
    const absoluteRoot = path.resolve(storage);

    expect(env.HOME).toBe(absoluteRoot);
    expect(env.ANTIGRAVITY_APP_DATA_DIR).toBe(absoluteRoot);
    expect(env.XDG_CONFIG_HOME).toBe(path.join(absoluteRoot, ".config"));
    expect(env.XDG_DATA_HOME).toBe(path.join(absoluteRoot, ".local", "share"));
    expect(env.DISPLAY).toBeDefined();
    expect(env.DBUS_SESSION_BUS_ADDRESS).toBeUndefined();
  });

  it("detects isolation when HOME matches ANTIGRAVITY_APP_DATA_DIR", () => {
    expect(
      isAntigravityStorageIsolation({
        HOME: "/data/antigravity",
        ANTIGRAVITY_APP_DATA_DIR: "/data/antigravity",
      }),
    ).toBe(true);
    expect(
      isAntigravityStorageIsolation({
        HOME: "/data/antigravity",
        ANTIGRAVITY_APP_DATA_DIR: "/other",
      }),
    ).toBe(false);
    expect(isAntigravityStorageIsolation(undefined)).toBe(false);
  });

  it("strips host DBUS and host XDG when merging isolated explicit env", () => {
    const explicit = buildIsolatedAntigravityProcessEnv(
      path.join(os.tmpdir(), "ag-merge-env-test"),
    );

    const merged = mergeAgentSpawnEnvironment(
      {
        PATH: "/usr/bin",
        LANG: "en_US.UTF-8",
        DBUS_SESSION_BUS_ADDRESS: "unix:path=/run/user/1000/bus",
        XDG_CONFIG_HOME: "/home/user/.config",
        XDG_DATA_HOME: "/home/user/.local/share",
        SECRET_LEAK: "must-not-appear",
      },
      explicit,
    );

    expect(merged.PATH).toBe(explicit.PATH);
    if (explicit.LANG !== undefined) {
      expect(merged.LANG).toBe(explicit.LANG);
    } else {
      expect(merged.LANG).toBe("en_US.UTF-8");
    }
    expect(merged.HOME).toBe(explicit.HOME);
    expect(merged.XDG_CONFIG_HOME).toBe(explicit.XDG_CONFIG_HOME);
    expect(merged.XDG_DATA_HOME).toBe(explicit.XDG_DATA_HOME);
    expect(merged.DBUS_SESSION_BUS_ADDRESS).toBeUndefined();
    expect(merged.SECRET_LEAK).toBeUndefined();
  });

  it("inherits host DBUS when explicit env is not isolated", () => {
    const merged = mergeAgentSpawnEnvironment(
      {
        DBUS_SESSION_BUS_ADDRESS: "unix:path=/run/user/1000/bus",
        PATH: "/usr/bin",
      },
      { HOME: "/home/user" },
    );

    expect(merged.DBUS_SESSION_BUS_ADDRESS).toBe("unix:path=/run/user/1000/bus");
    expect(merged.HOME).toBe("/home/user");
  });
});