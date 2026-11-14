import type { AgentExecutionInput } from "@agent-runtime-gateway/core";
import { imageBlock, textBlock, type ContentBlock } from "./acp/index.js";

export function promptBlocksFor(input: AgentExecutionInput): readonly ContentBlock[] {
  let messageText = input.message;
  if (input.toolChoice === "required") {
    messageText +=
      "\n\n[SYSTEM REQUIREMENT: You MUST invoke at least one of the available functions/tools to complete this request.]";
  } else if (isFunctionToolChoice(input.toolChoice)) {
    const requiredName = input.toolChoice.function.name;
    messageText += `\n\n[SYSTEM REQUIREMENT: You MUST call the specific tool '${requiredName}'.]`;
  }
  if (input.parallelToolCalls === false) {
    messageText +=
      "\n\n[SYSTEM REQUIREMENT: You must call at most one tool at a time. Parallel tool calling is disabled.]";
  }
  const blocks: ContentBlock[] = [textBlock(messageText)];
  for (const image of input.images ?? []) {
    blocks.push(imageBlock(image.mimeType, image.data));
  }
  return blocks;
}

function isFunctionToolChoice(
  toolChoice: AgentExecutionInput["toolChoice"],
): toolChoice is { readonly type: "function"; readonly function: { readonly name: string } } {
  return typeof toolChoice === "object" && toolChoice !== null && toolChoice.type === "function";
}
