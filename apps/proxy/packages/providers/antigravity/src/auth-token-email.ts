import type { AntigravityTokens } from "./types.js";

export function extractEmailFromTokens(tokens: AntigravityTokens): string | undefined {
  if (!tokens.id_token) return undefined;
  try {
    const parts = tokens.id_token.split(".");
    if (parts.length >= 2 && parts[1]) {
      const payloadJson = Buffer.from(parts[1], "base64url").toString("utf-8");
      const payload = JSON.parse(payloadJson) as { email?: unknown };
      return typeof payload.email === "string" ? payload.email : undefined;
    }
  } catch {
    // Ignore JWT parse failure
  }
  return undefined;
}

export async function fetchEmailFromUserInfo(
  accessToken: string,
): Promise<string | undefined> {
  try {
    const response = await fetch("https://www.googleapis.com/oauth2/v3/userinfo", {
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
    });
    if (!response.ok) return undefined;
    const data = (await response.json()) as { email?: unknown };
    return typeof data.email === "string" && data.email.length > 0 ? data.email : undefined;
  } catch {
    return undefined;
  }
}