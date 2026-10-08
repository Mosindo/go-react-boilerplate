import { ApiError, apiRequest, describeError } from "./client";
import { clearTokens, getAccessToken, saveTokens } from "../store/tokenStore";
import { onSessionExpired } from "./sessionEvents";

jest.mock("expo-secure-store", () => ({ getItemAsync: jest.fn(), setItemAsync: jest.fn(), deleteItemAsync: jest.fn() }));

const json = (status: number, body?: unknown) => ({
  ok: status >= 200 && status < 300,
  status,
  json: async () => body,
  text: async () => (body === undefined ? "" : JSON.stringify(body))
});

describe("apiRequest", () => {
  const fetchMock = jest.fn();
  beforeEach(async () => {
    fetchMock.mockReset();
    (global as unknown as { fetch: jest.Mock }).fetch = fetchMock;
    process.env.EXPO_PUBLIC_API_URL = "http://api.test";
    await clearTokens();
  });

  it("maps server error codes to friendly French messages", async () => {
    fetchMock.mockResolvedValueOnce(json(401, { error: "invalid credentials", code: "invalid_credentials" }));
    await expect(apiRequest("/auth/login", { method: "POST", body: {}, auth: false })).rejects.toMatchObject({
      status: 401,
      code: "invalid_credentials",
      message: "E-mail ou mot de passe incorrect."
    });
  });

  it("refreshes the token once on 401 and retries", async () => {
    await saveTokens({ accessToken: "old", refreshToken: "r1" });
    fetchMock
      .mockResolvedValueOnce(json(401, { error: "invalid token", code: "unauthorized" }))
      .mockResolvedValueOnce(json(200, { accessToken: "new", refreshToken: "r2" }))
      .mockResolvedValueOnce(json(200, { ok: true }));
    await expect(apiRequest("/me")).resolves.toEqual({ ok: true });
    expect(getAccessToken()).toBe("new");
    expect(fetchMock.mock.calls[2][1].headers.Authorization).toBe("Bearer new");
  });

  it("signs the user out when the refresh fails", async () => {
    await saveTokens({ accessToken: "old", refreshToken: "r1" });
    const expired = jest.fn();
    const off = onSessionExpired(expired);
    fetchMock
      .mockResolvedValueOnce(json(401, { code: "unauthorized" }))
      .mockResolvedValueOnce(json(401, { code: "invalid_refresh_token" }))
      .mockResolvedValue(json(401, {}));
    await expect(apiRequest("/me")).rejects.toBeInstanceOf(ApiError);
    expect(expired).toHaveBeenCalled();
    expect(getAccessToken()).toBeNull();
    off();
  });

  it("reports network failures clearly and hides 5xx details", async () => {
    fetchMock.mockRejectedValueOnce(new TypeError("Failed to fetch"));
    await expect(apiRequest("/me")).rejects.toMatchObject({ code: "network" });
    fetchMock.mockResolvedValueOnce(json(500, { error: "pq: secret detail", code: "internal" }));
    await expect(apiRequest("/me")).rejects.toThrow(/serveur rencontre un problème/);
    expect(describeError(null)).toMatch(/erreur/i);
  });
});
