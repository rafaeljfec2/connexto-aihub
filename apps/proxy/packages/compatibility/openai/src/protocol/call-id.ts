import { createHash } from "node:crypto";

export const MAX_RESPONSES_CALL_ID_LENGTH = 64;

export function responsesCallId(raw: string): string {
  if (raw.length <= MAX_RESPONSES_CALL_ID_LENGTH) {
    return raw;
  }
  const digest = createHash("sha256").update(raw).digest("hex").slice(0, 32);
  return `call_${digest}`;
}
