import {
  buildPreferences,
  buildProfileInput,
  initialDraft,
  validateStep,
  type Draft
} from "../src/features/onboarding/draft";
import { moveId, validatePickedImage } from "../src/features/profile/PhotoManager";
import { toggleInList } from "../src/features/profile/options";

jest.mock("expo-image-picker", () => ({}));
jest.mock("expo-location", () => ({}));

const NOW = new Date(2026, 5, 15);
const valid: Draft = {
  ...initialDraft,
  firstName: " Ana ",
  birth: { day: "01", month: "02", year: "1995" },
  gender: "woman",
  interestedIn: ["man"],
  city: " Lisbon ",
  bio: " hi ",
  interests: ["hiking"]
};

describe("validateStep", () => {
  it("requires each step's data", () => {
    expect(validateStep("name", initialDraft, NOW)).not.toBeNull();
    expect(validateStep("birthDate", initialDraft, NOW)).not.toBeNull();
    expect(validateStep("gender", initialDraft, NOW)).not.toBeNull();
    expect(validateStep("preferences", initialDraft, NOW)).not.toBeNull();
    expect(validateStep("location", initialDraft, NOW)).not.toBeNull();
  });
  it("blocks minors at the birth date step", () => {
    const minor = { ...valid, birth: { day: "01", month: "01", year: "2015" } };
    expect(validateStep("birthDate", minor, NOW)).not.toBeNull();
  });
  it("accepts a complete draft on every step", () => {
    for (const step of [
      "name",
      "birthDate",
      "gender",
      "preferences",
      "location",
      "about",
      "photos"
    ] as const) {
      expect(validateStep(step, valid, NOW)).toBeNull();
    }
  });
  it("accepts coordinates without a city", () => {
    expect(
      validateStep("location", { ...valid, city: "", coords: { latitude: 1, longitude: 2 } }, NOW)
    ).toBeNull();
  });
});

describe("request builders", () => {
  it("trims and shapes the profile payload", () => {
    expect(buildProfileInput(valid)).toEqual({
      firstName: "Ana",
      birthDate: "1995-02-01",
      gender: "woman",
      bio: "hi",
      city: "Lisbon",
      interests: ["hiking"]
    });
  });
  it("returns null when required fields are missing", () => {
    expect(buildProfileInput(initialDraft)).toBeNull();
  });
  it("seeds visible, distance-sharing preferences", () => {
    expect(buildPreferences(valid)).toMatchObject({
      interestedIn: ["man"],
      discoverable: true,
      showDistance: true
    });
  });
});

describe("photo helpers", () => {
  it("moves ids and ignores out of range moves", () => {
    expect(moveId(["a", "b", "c"], "c", 0)).toEqual(["c", "a", "b"]);
    expect(moveId(["a", "b", "c"], "a", 1)).toEqual(["b", "a", "c"]);
    expect(moveId(["a", "b"], "a", 5)).toEqual(["a", "b"]);
    expect(moveId(["a", "b"], "z", 0)).toEqual(["a", "b"]);
  });
  it("rejects unsupported and oversized images", () => {
    expect(validatePickedImage({ mimeType: "image/gif" })).not.toBeNull();
    expect(validatePickedImage({ mimeType: "image/jpeg", fileSize: 6 * 1024 * 1024 })).not.toBeNull();
    expect(validatePickedImage({ mimeType: "image/png", fileSize: 1024 })).toBeNull();
    expect(validatePickedImage({})).toBeNull();
  });
  it("toggles list membership with a cap", () => {
    expect(toggleInList(["a"], "b")).toEqual(["a", "b"]);
    expect(toggleInList(["a", "b"], "a")).toEqual(["b"]);
    expect(toggleInList(["a", "b"], "c", 2)).toEqual(["a", "b"]);
  });
});
