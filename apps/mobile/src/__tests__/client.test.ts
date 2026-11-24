import { ApiError, request, setFetchImplementation } from "../api/client";
import { clearTokens, getTokens, setTokens } from "../auth/session";
import { errorMessage } from "../utils/errors";

jest.mock("expo-secure-store", () => {
  const store = new Map<string, string>();
  return {
    getItemAsync: jest.fn(async (k: string) => store.get(k) ?? null),
    setItemAsync: jest.fn(async (k: string, v: string) => void store.set(k, v)),
    deleteItemAsync: jest.fn(async (k: string) => void store.delete(k))
  };
});

function json(status: number, body: unknown): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("api client", () => {
  beforeEach(async () => {
    await setTokens({ accessToken: "old-access", refreshToken: "refresh-1" });
  });

  it("refreshes once for concurrent 401s and retries with the new token", async () => {
    let refreshCalls = 0;
    const seen: string[] = [];
    setFetchImplementation(async (input, init) => {
      const url = String(input);
      const auth = new Headers(init?.headers).get("Authorization") ?? "";
      if (url.endsWith("/auth/refresh")) {
        refreshCalls += 1;
        await new Promise((r) => setTimeout(r, 10));
        return json(200, { accessToken: "new-access", refreshToken: "refresh-2", expiresIn: 900, user: { id: "u", email: "e", createdAt: "" } });
      }
      seen.push(auth);
      return auth === "Bearer new-access" ? json(200, { ok: true }) : json(401, { error: "expired", code: "invalid_token" });
    });

    const results = await Promise.all([request("/a"), request("/b"), request("/c")]);
    expect(results).toEqual([{ ok: true }, { ok: true }, { ok: true }]);
    expect(refreshCalls).toBe(1);
    expect(getTokens()).toEqual({ accessToken: "new-access", refreshToken: "refresh-2" });
    expect(seen.filter((h) => h === "Bearer new-access")).toHaveLength(3);
  });

  it("signs out when the refresh token is rejected", async () => {
    setFetchImplementation(async (input) =>
      String(input).endsWith("/auth/refresh") ? json(401, { error: "nope", code: "invalid_refresh_token" }) : json(401, { error: "expired", code: "invalid_token" })
    );
    await expect(request("/me")).rejects.toBeInstanceOf(ApiError);
    expect(getTokens()).toBeNull();
  });

  it("maps server error codes to French messages", async () => {
    setFetchImplementation(async () => json(409, { error: "already", code: "email_taken" }));
    const error = await request("/auth/register", { method: "POST", auth: false }).catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(409);
    expect(errorMessage(error)).toBe("Un compte existe déjà avec cet e-mail.");
  });

  it("reports network failures with a dedicated code", async () => {
    setFetchImplementation(async () => {
      throw new TypeError("Network request failed");
    });
    const error = (await request("/me").catch((e: unknown) => e)) as ApiError;
    expect(error.code).toBe("network");
    await clearTokens();
  });
});
