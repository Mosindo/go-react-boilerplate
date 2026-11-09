import { isValidEmail, passwordProblem } from "./validation";

describe("validation", () => {
  it("accepts and rejects emails", () => {
    expect(isValidEmail(" a@b.co ")).toBe(true);
    expect(isValidEmail("nope")).toBe(false);
    expect(isValidEmail("a b@c.de")).toBe(false);
  });
  it("checks password length", () => {
    expect(passwordProblem("short")).not.toBeNull();
    expect(passwordProblem("long-enough-1")).toBeNull();
  });
});
