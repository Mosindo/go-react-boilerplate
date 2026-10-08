import { parseTokens } from "./tokenStore";

describe("parseTokens", () => {
  it("restores a valid pair and rejects corrupted storage", () => {
    expect(parseTokens('{"accessToken":"a","refreshToken":"r"}')).toEqual({ accessToken: "a", refreshToken: "r" });
    expect(parseTokens('{"accessToken":"a"}')).toBeNull();
    expect(parseTokens("{broken")).toBeNull();
    expect(parseTokens(null)).toBeNull();
  });
});
