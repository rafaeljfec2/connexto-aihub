import type { AntigravityTokens, AntigravityViabilityGate } from "./types.js";

export const GOOGLE_OAUTH_CLIENT_ID = [
  "1071006060591",
  "tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com",
].join("-");

export const GOOGLE_OAUTH_CLIENT_SECRET = [
  "GOCSPX",
  "K58FWR486LdLJ1mLB8sXC4z6qDAf",
].join("-");

export const GOOGLE_OAUTH_REDIRECT_URI = "https://antigravity.google/oauth-callback";

export const GOOGLE_OAUTH_SCOPES = [
  "https://www.googleapis.com/auth/cloud-platform",
  "https://www.googleapis.com/auth/userinfo.email",
  "https://www.googleapis.com/auth/userinfo.profile",
  "https://www.googleapis.com/auth/cclog",
  "https://www.googleapis.com/auth/experimentsandconfigs",
  "https://www.googleapis.com/auth/aicode",
  "openid",
].join(" ");

export const ANTIGRAVITY_VIABILITY_GATE: AntigravityViabilityGate = "PARTIALLY_SUPPORTED";

export const ANTIGRAVITY_VIABILITY_LIMITATIONS: readonly string[] = [
  "Google Antigravity personal OAuth credentials are bound to Google internal client ID and CLI local storage.",
  "Exporting personal access or refresh tokens is officially NOT_SUPPORTED by Google CLI/SDK.",
  "Delegated token transfer from AI Hub to Proxy via HTTP headers is officially NOT_SUPPORTED by Google.",
  "Official google-antigravity SDK only supports Gemini API Key and Vertex AI (ADC), rejecting personal OAuth tokens.",
];

export function buildGoogleOAuthAuthorizationUrl(params: {
  readonly redirectUri: string;
  readonly state: string;
  readonly codeChallenge: string;
}): string {
  const url = new URL("https://accounts.google.com/o/oauth2/auth");
  url.searchParams.set("access_type", "offline");
  url.searchParams.set("client_id", GOOGLE_OAUTH_CLIENT_ID);
  url.searchParams.set("code_challenge", params.codeChallenge);
  url.searchParams.set("code_challenge_method", "S256");
  url.searchParams.set("prompt", "consent");
  url.searchParams.set("redirect_uri", params.redirectUri);
  url.searchParams.set("response_type", "code");
  url.searchParams.set("scope", GOOGLE_OAUTH_SCOPES);
  url.searchParams.set("state", params.state);
  return url.toString();
}

export async function defaultGoogleTokenExchanger(params: {
  readonly clientId: string;
  readonly clientSecret?: string;
  readonly code: string;
  readonly codeVerifier: string;
  readonly redirectUri: string;
}): Promise<AntigravityTokens> {
  const body = new URLSearchParams({
    client_id: params.clientId,
    client_secret: params.clientSecret ?? GOOGLE_OAUTH_CLIENT_SECRET,
    code: params.code,
    code_verifier: params.codeVerifier,
    redirect_uri: params.redirectUri,
    grant_type: "authorization_code",
  });

  const response = await fetch("https://oauth2.googleapis.com/token", {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: body.toString(),
  });

  if (!response.ok) {
    const errorText = await response.text().catch(() => "");
    throw new Error(`Google token exchange failed (${response.status}): ${errorText}`);
  }

  return (await response.json()) as AntigravityTokens;
}