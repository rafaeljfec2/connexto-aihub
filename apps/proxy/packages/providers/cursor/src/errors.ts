import {
  AuthenticationError,
  ConfigurationError,
  CursorAgentError,
  CursorSdkError,
  RateLimitError,
} from "@cursor/sdk";

export interface NormalizedCursorError {
  message: string;
  code: string;
  retryable: boolean;
  cause?: unknown;
}

/** Cursor upstream signals transient overload with capacity / unavailable wording. */
export function isCursorCapacityErrorMessage(message: string): boolean {
  const normalized = message.trim().toLowerCase();
  if (normalized.includes("server at capacity")) return true;
  if (normalized.includes("at capacity")) return true;
  if (normalized.includes("overloaded")) return true;
  if (normalized.startsWith("[unavailable]")) return true;
  return false;
}

export function normalizeCursorError(error: unknown): NormalizedCursorError {
  if (error instanceof ConfigurationError) {
    return {
      message: error.message,
      code: "cursor_configuration_error",
      retryable: false,
      cause: error,
    };
  }

  if (error instanceof AuthenticationError) {
    return {
      message: error.message,
      code: "cursor_authentication_error",
      retryable: false,
      cause: error,
    };
  }

  if (error instanceof RateLimitError) {
    return {
      message: error.message,
      code: "cursor_rate_limit",
      retryable: true,
      cause: error,
    };
  }

  if (error instanceof CursorAgentError) {
    const capacity = isCursorCapacityErrorMessage(error.message);
    return {
      message: error.message,
      code: capacity ? "cursor_rate_limit" : "cursor_agent_error",
      retryable: capacity,
      cause: error,
    };
  }

  if (error instanceof CursorSdkError) {
    const capacity = isCursorCapacityErrorMessage(error.message);
    return {
      message: error.message,
      code: capacity ? "cursor_rate_limit" : "cursor_sdk_error",
      retryable: capacity,
      cause: error,
    };
  }

  if (error instanceof Error) {
    const capacity = isCursorCapacityErrorMessage(error.message);
    return {
      message: error.message,
      code: capacity ? "cursor_rate_limit" : "cursor_unknown_error",
      retryable: capacity,
      cause: error,
    };
  }

  return {
    message: "Cursor provider failed with an unknown error",
    code: "cursor_unknown_error",
    retryable: false,
    cause: error,
  };
}
