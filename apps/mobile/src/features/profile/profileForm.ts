import type { MyProfile } from "../../api/types";
import { ageFromIso, toIsoDate } from "../../utils/format";
import type { BasicsValue } from "./components/BasicsForm";

export function basicsFromProfile(profile?: MyProfile): BasicsValue {
  const [year, month, day] = profile?.birthdate ? profile.birthdate.split("-") : ["", "", ""];
  return {
    firstName: profile?.firstName ?? "",
    day: day ?? "",
    month: month ?? "",
    year: year ?? "",
    gender: profile?.gender ?? "",
    city: profile?.city ?? ""
  };
}

export type BasicsErrors = { firstName?: string | null; birthdate?: string | null; gender?: string | null };

/** Client-side validation mirroring the API rules (the API stays the authority). */
export function validateBasics(value: BasicsValue): { errors: BasicsErrors; birthdate: string | null } {
  const errors: BasicsErrors = {};
  const name = value.firstName.trim();
  if (!name) {
    errors.firstName = "Indique ton prénom.";
  } else if (!/^[\p{L}\p{M}' ’-]+$/u.test(name)) {
    errors.firstName = "Lettres, espaces, tirets et apostrophes uniquement.";
  }
  const birthdate = toIsoDate(value.day, value.month, value.year);
  if (!birthdate) {
    errors.birthdate = "Date invalide.";
  } else if (ageFromIso(birthdate) < 18) {
    errors.birthdate = "Lueur est réservée aux personnes majeures.";
  } else if (ageFromIso(birthdate) > 120) {
    errors.birthdate = "Date invalide.";
  }
  if (!value.gender) {
    errors.gender = "Choisis une option.";
  }
  return { errors, birthdate };
}

export function hasErrors(errors: BasicsErrors): boolean {
  return Object.values(errors).some(Boolean);
}
