/** Extracts Chat Completions `role: tool` messages after the latest assistant tool_calls block. */
export function extractActiveChatToolResults(
  messages: readonly unknown[],
): { toolCallId: string; toolName?: string; content: string }[] {
  let lastAssistantWithToolsIdx = -1;
  for (let i = messages.length - 1; i >= 0; i--) {
    const m = messages[i];
    if (
      m &&
      typeof m === "object" &&
      (m as Record<string, unknown>).role === "assistant" &&
      Array.isArray((m as Record<string, unknown>).tool_calls) &&
      ((m as Record<string, unknown>).tool_calls as unknown[]).length > 0
    ) {
      lastAssistantWithToolsIdx = i;
      break;
    }
  }

  const activeMessages =
    lastAssistantWithToolsIdx >= 0
      ? messages.slice(lastAssistantWithToolsIdx + 1)
      : messages;

  return activeMessages
    .filter(
      (m): m is Record<string, unknown> =>
        Boolean(m) &&
        typeof m === "object" &&
        (m as Record<string, unknown>).role === "tool",
    )
    .map((m) => ({
      toolCallId: typeof m.tool_call_id === "string" ? m.tool_call_id : "",
      toolName: typeof m.name === "string" ? m.name : undefined,
      content:
        typeof m.content === "string"
          ? m.content
          : JSON.stringify(m.content ?? ""),
    }));
}
