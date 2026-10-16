import { describe, expect, it } from "vitest";
import { singleFlight } from "../singleFlight";

describe("singleFlight", () => {
  it("shares one run between concurrent callers, then allows a new one", async () => {
    let runs = 0;
    const call = singleFlight(async () => {
      runs += 1;
      await new Promise((resolve) => setTimeout(resolve, 5));
      return runs;
    });

    const [a, b, c] = await Promise.all([call(), call(), call()]);
    expect([a, b, c]).toEqual([1, 1, 1]);
    expect(await call()).toBe(2);
  });

  it("releases the slot after a failure", async () => {
    let attempts = 0;
    const call = singleFlight(async () => {
      attempts += 1;
      if (attempts === 1) {
        throw new Error("boom");
      }
      return "ok";
    });
    await expect(call()).rejects.toThrow("boom");
    await expect(call()).resolves.toBe("ok");
  });
});
