import { formToInput, emptyProfileForm, profileToForm, validatePreferences } from "./profileForm";

describe("profileForm", () => {
  it("produces a clean API payload", () => {
    const { input, errors } = formToInput({
      ...emptyProfileForm(),
      firstName: "  Anne   Sophie ",
      birthdate: "02/04/1995",
      gender: "woman",
      bio: " Hello ",
      heightCm: "168",
      interestIds: [1, 2]
    });
    expect(errors).toEqual({});
    expect(input).toEqual({
      firstName: "Anne Sophie",
      birthdate: "1995-04-02",
      gender: "woman",
      bio: "Hello",
      jobTitle: null,
      city: null,
      heightCm: 168,
      lookingFor: null,
      interestIds: [1, 2]
    });
  });

  it("reports validation errors", () => {
    const { input, errors } = formToInput({ ...emptyProfileForm(), firstName: "<b>", birthdate: "01/01/2015", heightCm: "300" });
    expect(input).toBeUndefined();
    expect(Object.keys(errors).sort()).toEqual(["birthdate", "firstName", "gender", "heightCm"]);
  });

  it("round-trips an existing profile", () => {
    const form = profileToForm({
      firstName: "Léa",
      birthdate: "1994-05-01",
      age: 32,
      gender: "woman",
      bio: "",
      jobTitle: null,
      city: "Lyon",
      heightCm: null,
      lookingFor: "relationship",
      interests: [{ id: 3, slug: "x", label: "X" }],
      hasLocation: true,
      locationUpdatedAt: null
    });
    expect(form.birthdate).toBe("01/05/1994");
    expect(form.interestIds).toEqual([3]);
  });

  it("validates preferences", () => {
    expect(validatePreferences({ interestedIn: [], minAge: 18, maxAge: 30, maxDistanceKm: 10 })).not.toBeNull();
    expect(validatePreferences({ interestedIn: ["man"], minAge: 40, maxAge: 30, maxDistanceKm: 10 })).not.toBeNull();
    expect(validatePreferences({ interestedIn: ["man"], minAge: 20, maxAge: 30, maxDistanceKm: 10 })).toBeNull();
  });
});
