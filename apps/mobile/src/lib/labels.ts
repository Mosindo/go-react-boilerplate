import type { Gender, ReportReason } from "../api/types";

export const GENDER_LABELS: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  non_binary: "Non-binaire"
};

export const INTERESTED_IN_LABELS: Record<Gender, string> = {
  woman: "Des femmes",
  man: "Des hommes",
  non_binary: "Des personnes non-binaires"
};

export const GENDERS: Gender[] = ["woman", "man", "non_binary"];

export const REPORT_REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "harassment", label: "Harcèlement" },
  { value: "scam", label: "Arnaque" },
  { value: "spam", label: "Spam" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

/** The API rounds distances to 5 km steps, so the wording stays an upper bound. */
export function distanceLabel(km: number | undefined): string | null {
  if (km === undefined) return null;
  return km <= 5 ? "À moins de 5 km" : `À moins de ${km} km`;
}
