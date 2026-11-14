import crypto from "node:crypto";
import fs from "node:fs/promises";
import path from "node:path";
import { extractEmailFromTokens } from "./auth-token-email.js";
import type { AntigravityAccountInfo, AntigravityTokens } from "./types.js";

export function credentialFilePaths(homeDir: string): string[] {
  return [
    path.join(homeDir, ".gemini", "oauth_creds.json"),
    path.join(homeDir, ".gemini", "jetski-standalone-oauth-token"),
    path.join(homeDir, ".gemini", "google_accounts.json"),
    path.join(homeDir, ".gemini", "antigravity-cli", "antigravity-oauth-token"),
    path.join(homeDir, ".config", "antigravity", "credential-user-scope"),
  ];
}

export async function detectAccountFromHomeDir(
  homeDir: string,
): Promise<AntigravityAccountInfo> {
  try {
    const accountsFile = path.join(homeDir, ".gemini", "google_accounts.json");
    const content = await fs.readFile(accountsFile, "utf8");
    const data = JSON.parse(content) as { active?: unknown };
    if (typeof data.active === "string" && data.active.length > 0) {
      return { authenticated: true, account: data.active };
    }
  } catch {
    try {
      const tokenFile = path.join(homeDir, ".gemini", "jetski-standalone-oauth-token");
      const content = await fs.readFile(tokenFile, "utf8");
      const data = JSON.parse(content) as { id_token_email?: unknown };
      if (typeof data.id_token_email === "string" && data.id_token_email.length > 0) {
        return { authenticated: true, account: data.id_token_email };
      }
    } catch {
      try {
        const credsFile = path.join(homeDir, ".gemini", "oauth_creds.json");
        const content = await fs.readFile(credsFile, "utf8");
        const data = JSON.parse(content) as { id_token?: unknown; access_token?: unknown };
        let email: string | undefined;
        if (typeof data.id_token === "string" && data.id_token.length > 0) {
          email = extractEmailFromTokens({
            id_token: data.id_token,
            access_token: typeof data.access_token === "string" ? data.access_token : "",
            token_type: "Bearer",
          });
        }
        if (typeof data.access_token === "string" && data.access_token.length > 0) {
          return { authenticated: true, account: email };
        }
      } catch {
        // No account found
      }
    }
  }

  return { authenticated: false };
}

export async function persistLocalCredentials(
  homeDir: string,
  tokens: AntigravityTokens,
  email?: string,
): Promise<void> {
  const geminiDir = path.join(homeDir, ".gemini");
  const configDir = path.join(homeDir, ".config", "antigravity");

  try {
    await fs.mkdir(geminiDir, { recursive: true, mode: 0o700 });
    await fs.mkdir(configDir, { recursive: true, mode: 0o700 });

    const expiryDate =
      tokens.expiry_date ?? Date.now() + (tokens.expires_in ?? 3600) * 1000;

    const credsPath = path.join(geminiDir, "oauth_creds.json");
    const credsData = {
      access_token: tokens.access_token,
      scope: tokens.scope ?? "",
      token_type: tokens.token_type ?? "Bearer",
      id_token: tokens.id_token ?? "",
      expiry_date: expiryDate,
      refresh_token: tokens.refresh_token ?? "",
    };
    await fs.writeFile(credsPath, JSON.stringify(credsData, null, 2), { mode: 0o600 });

    if (email) {
      const accountsPath = path.join(geminiDir, "google_accounts.json");
      let existingAccounts = { active: email, old: [] as string[] };
      try {
        const raw = await fs.readFile(accountsPath, "utf-8");
        const parsed = JSON.parse(raw) as { active?: string; old?: string[] };
        const oldList = Array.isArray(parsed.old) ? parsed.old : [];
        if (parsed.active && parsed.active !== email && !oldList.includes(parsed.active)) {
          oldList.push(parsed.active);
        }
        existingAccounts = { active: email, old: oldList };
      } catch {
        // New file
      }
      await fs.writeFile(accountsPath, JSON.stringify(existingAccounts, null, 2), {
        mode: 0o600,
      });
    }

    const expiresInSeconds =
      tokens.expires_in ?? Math.max(0, Math.floor((expiryDate - Date.now()) / 1000));
    const expiryIso = new Date(expiryDate).toISOString();

    const cliOAuthPayload = {
      token: {
        access_token: tokens.access_token,
        refresh_token: tokens.refresh_token ?? "",
        token_type: tokens.token_type ?? "Bearer",
        expiry: expiryIso,
        expires_in: expiresInSeconds,
      },
      auth_method: "oauth",
      id_token_email: email,
    };

    const jetskiPath = path.join(geminiDir, "jetski-standalone-oauth-token");
    await fs.writeFile(jetskiPath, JSON.stringify(cliOAuthPayload, null, 2), { mode: 0o600 });

    const agCliDir = path.join(geminiDir, "antigravity-cli");
    await fs.mkdir(agCliDir, { recursive: true, mode: 0o700 });
    const agCliTokenPath = path.join(agCliDir, "antigravity-oauth-token");
    await fs.writeFile(agCliTokenPath, JSON.stringify(cliOAuthPayload, null, 2), {
      mode: 0o600,
    });

    const scopePath = path.join(configDir, "credential-user-scope");
    try {
      await fs.access(scopePath);
    } catch {
      const randomScope = "u" + crypto.randomBytes(8).toString("hex");
      await fs.writeFile(scopePath, randomScope, { mode: 0o600 });
    }
  } catch {
    // In restricted filesystem environments, do not fail the auth flow
  }
}