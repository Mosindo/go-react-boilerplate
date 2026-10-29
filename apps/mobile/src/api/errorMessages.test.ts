import { translateApiError } from "./errorMessages";

describe("translateApiError", () => {
  it("translates known messages", () => {
    expect(translateApiError("invalid email or password")).toBe(
      "Adresse e-mail ou mot de passe incorrect."
    );
    expect(translateApiError("you must be at least 18 years old to use this app")).toContain(
      "18 ans"
    );
    expect(translateApiError("you can upload at most 6 photos")).toContain("6 photos");
  });

  it("keeps unknown messages untouched", () => {
    expect(translateApiError("something new")).toBe("something new");
  });
});
