import { cursorPublicRaw, defaultCursorDeps, defaultCursorEnv } from "@agent-runtime-gateway/provider-cursor";

/** POST the original Responses JSON to Cursor's OpenAI-compatible endpoint. */
export async function postCursorResponsesPassthrough(
  apiKey: string,
  body: Record<string, unknown>,
  signal?: AbortSignal,
): Promise<globalThis.Response> {
  const env = defaultCursorEnv();
  const deps = defaultCursorDeps;
  return cursorPublicRaw(env, deps, apiKey, "/v1/responses", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal,
  });
}
