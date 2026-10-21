export type Gender = "man" | "woman" | "non_binary";
export type LookingFor = "long_term" | "short_term" | "friendship" | "open" | "unsure";

export const GENDERS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Femme" },
  { value: "man", label: "Homme" },
  { value: "non_binary", label: "Non-binaire" }
];

export const INTERESTED_IN: { value: Gender; label: string }[] = [
  { value: "woman", label: "Des femmes" },
  { value: "man", label: "Des hommes" },
  { value: "non_binary", label: "Des personnes non-binaires" }
];

export const LOOKING_FOR: { value: LookingFor; label: string }[] = [
  { value: "long_term", label: "Relation sérieuse" },
  { value: "short_term", label: "Sans prise de tête" },
  { value: "friendship", label: "Amitié" },
  { value: "open", label: "Ouvert·e" },
  { value: "unsure", label: "Je ne sais pas encore" }
];

export const REPORT_REASONS: { value: string; label: string }[] = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "underage", label: "Personne mineure" },
  { value: "spam", label: "Spam" },
  { value: "scam", label: "Arnaque" },
  { value: "other", label: "Autre" }
];

export function labelOf<T extends string>(list: { value: T; label: string }[], value: string): string {
  return list.find((item) => item.value === value)?.label ?? value;
}

export const DISTANCE_STEPS: (number | null)[] = [5, 10, 25, 50, 100, 250, null];

export function distanceLabel(km: number | null): string {
  return km === null ? "Partout" : `${km} km`;
}
