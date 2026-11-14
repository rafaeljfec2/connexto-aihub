import { persistLocalCredentials } from "./auth-credential-store.js";
import { extractEmailFromTokens, fetchEmailFromUserInfo } from "./auth-token-email.js";
import {
  AuthorizationDeniedError,
  InvalidStateError,
  OperationExpiredError,
  OperationNotFoundError,
} from "./errors.js";
import { GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET } from "./google-oauth-config.js";
import type {
  AntigravityAuthCallbackResult,
  AntigravityAuthManagerOptions,
  AntigravityAuthOperation,
  AntigravityTokens,
} from "./types.js";

export async function processAuthCallback(params: {
  readonly homeDir: string;
  readonly operations: Map<string, AntigravityAuthOperation>;
  readonly stateToOperationId: Map<string, string>;
  readonly tokenExchanger: NonNullable<AntigravityAuthManagerOptions["tokenExchanger"]>;
  readonly hasExplicitTokenExchanger: boolean;
  readonly onTokensPersisted?: (
    tokens: AntigravityTokens,
    email?: string,
  ) => Promise<void>;
  readonly accountDetector: () => Promise<{ authenticated: boolean; account?: string }>;
  readonly callback: {
    state?: string;
    code?: string;
    error?: string;
    error_description?: string;
  };
}): Promise<AntigravityAuthCallbackResult> {
  const rawState = params.callback.state?.trim();
  if (!rawState) {
    throw new InvalidStateError("Missing required auth state parameter.");
  }

  const operationId = params.stateToOperationId.get(rawState);
  if (!operationId) {
    throw new InvalidStateError("No active auth operation matches the provided state.");
  }

  const operation = params.operations.get(operationId);
  if (!operation) {
    throw new OperationNotFoundError(operationId);
  }

  const isExpired =
    operation.status === "expired" ||
    Date.now() > new Date(operation.expiresAt).getTime();

  if (isExpired) {
    params.operations.set(operationId, { ...operation, status: "expired" });
    throw new OperationExpiredError(operationId);
  }

  if (params.callback.error) {
    const sanitizedError =
      params.callback.error_description?.trim() || params.callback.error.trim();
    params.operations.set(operationId, {
      ...operation,
      status: "failed",
      error: sanitizedError,
    });
    throw new AuthorizationDeniedError(sanitizedError);
  }

  const rawCode = params.callback.code?.trim();
  if (!rawCode) {
    throw new InvalidStateError("Missing authorization code in callback.");
  }

  let detectedAccount: string | undefined;

  if (operation.codeVerifier && operation.redirectUri) {
    try {
      const tokens = await params.tokenExchanger({
        clientId: GOOGLE_OAUTH_CLIENT_ID,
        clientSecret: GOOGLE_OAUTH_CLIENT_SECRET,
        code: rawCode,
        codeVerifier: operation.codeVerifier,
        redirectUri: operation.redirectUri,
      });

      let email = extractEmailFromTokens(tokens);
      if (!email && tokens.access_token) {
        email = await fetchEmailFromUserInfo(tokens.access_token).catch(() => undefined);
      }

      if (email) {
        detectedAccount = email;
      }

      await persistLocalCredentials(params.homeDir, tokens, email);
      if (params.onTokensPersisted) {
        await params.onTokensPersisted(tokens, email);
      }
    } catch (err) {
      if (params.hasExplicitTokenExchanger) {
        throw new AuthorizationDeniedError(
          `Failed to exchange authorization code: ${err instanceof Error ? err.message : String(err)}`,
        );
      }
    }
  }

  if (!detectedAccount) {
    const accountInfo = await params.accountDetector().catch(() => ({
      authenticated: true,
      account: undefined,
    }));
    detectedAccount = accountInfo.account;
  }

  params.operations.set(operationId, {
    ...operation,
    status: "completed",
    account: detectedAccount,
  });

  return {
    operationId,
    status: "completed",
    account: detectedAccount,
    message: "Antigravity authentication completed successfully.",
  };
}