import { describe, expect, it } from "vitest";
import type { MyProfile } from "../api/profile";
import {
  DEFAULT_PRIVACY_VALUES,
  isProfileDirty,
  profileToFormValues,
  profileToPrivacy,
  toProfileInput
} from "./profileForm";

const profile: MyProfile = {
  userId: "u1",
  firstName: "Sam",
  age: 30,
  gender: "man",
  bio: "Hi",
  city: "Lyon",
  distanceKm: null,
  interests: ["hiking", "jazz"],
  photos: [],
  hasLocation: true,
  showDistance: false,
  showAge: true,
  discoverable: false,
  isComplete: true
};

describe("profile form helpers", () => {
  it("maps a profile to form values and privacy", () => {
    expect(profileToFormValues(profile)).toEqual({
      firstName: "Sam",
      gender: "man",
      bio: "Hi",
      city: "Lyon",
      interests: ["hiking", "jazz"]
    });
    expect(profileToPrivacy(profile)).toEqual({
      showDistance: false,
      showAge: true,
      discoverable: false
    });
    expect(profileToPrivacy(null)).toEqual(DEFAULT_PRIVACY_VALUES);
    expect(profileToFormValues(null).interests).toEqual([]);
  });

  it("builds a trimmed request body", () => {
    const input = toProfileInput(
      { firstName: " Sam ", gender: "man", bio: " hello ", city: " Lyon ", interests: ["a"] },
      DEFAULT_PRIVACY_VALUES
    );
    expect(input).toEqual({
      firstName: "Sam",
      gender: "man",
      bio: "hello",
      city: "Lyon",
      interests: ["a"],
      showDistance: true,
      showAge: true,
      discoverable: true
    });
  });

  it("refuses to build a body without a gender", () => {
    expect(() =>
      toProfileInput(
        { firstName: "A", gender: null, bio: "", city: "", interests: [] },
        DEFAULT_PRIVACY_VALUES
      )
    ).toThrow();
  });

  it("detects changes ignoring whitespace and order", () => {
    const base = profileToFormValues(profile);
    expect(isProfileDirty(base, { ...base, firstName: " Sam " })).toBe(false);
    expect(isProfileDirty(base, { ...base, interests: ["jazz", "hiking"] })).toBe(false);
    expect(isProfileDirty(base, { ...base, interests: ["jazz"] })).toBe(true);
    expect(isProfileDirty(base, { ...base, bio: "Hello" })).toBe(true);
  });
});
