import { apiRequest, ApiError } from "../api/client";
import { getTokens, saveTokens } from "../store/tokenStore";

type FetchMock = jest.Mock<Promise<Response>, [string, RequestInit?]>;

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("apiRequest", () => {
  const originalFetch = globalThis.fetch;
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("refreshes the access token once and retries on 401, single flight", async () => {
    process.env.EXPO_PUBLIC_API_URL = "http://api.test";
    await saveTokens({ accessToken: "old", refreshToken: "r1" });
    const fetchMock: FetchMock = jest.fn(async (url: string, init?: RequestInit) => {
      const auth = (init?.headers as Record<string, string> | undefined)?.Authorization;
      if (url.endsWith("/auth/refresh")) {
        return json(200, { accessToken: "new", refreshToken: "r2", user: {} });
      }
      return auth === "Bearer new" ? json(200, { ok: true }) : json(401, { error: "invalid token" });
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    const [a, b] = await Promise.all([
      apiRequest<{ ok: boolean }>("/me"),
      apiRequest<{ ok: boolean }>("/me")
    ]);
    expect(a.ok && b.ok).toBe(true);
    expect(fetchMock.mock.calls.filter(([u]) => u.endsWith("/auth/refresh"))).toHaveLength(1);
    expect((await getTokens())?.refreshToken).toBe("r2");
  });

  it("surfaces API error messages", async () => {
    await saveTokens({ accessToken: "t", refreshToken: "r" });
    globalThis.fetch = jest.fn(async () => json(409, { error: "already swiped" })) as unknown as typeof fetch;
    await expect(apiRequest("/swipes", { method: "POST", body: {} })).rejects.toMatchObject({
      status: 409,
      message: "already swiped"
    });
    await expect(apiRequest("/swipes", { method: "POST", body: {} })).rejects.toBeInstanceOf(ApiError);
  });
});
