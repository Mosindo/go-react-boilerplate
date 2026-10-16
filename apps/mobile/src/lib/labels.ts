import type { Gender, LookingFor, ReportReason } from "../api/types";

export const GENDERS: readonly { value: Gender; label: string; plural: string }[] = [
  { value: "woman", label: "Une femme", plural: "Des femmes" },
  { value: "man", label: "Un homme", plural: "Des hommes" },
  { value: "nonbinary", label: "Non-binaire", plural: "Des personnes non-binaires" }
];

export const LOOKING_FOR: readonly { value: LookingFor; label: string }[] = [
  { value: "relationship", label: "Une relation" },
  { value: "casual", label: "Quelque chose de décontracté" },
  { value: "friendship", label: "De l'amitié" },
  { value: "open", label: "Je ne sais pas encore" }
];

export const DISTANCE_CHOICES = [5, 10, 25, 50, 100, 250, 500] as const;

export function lookingForLabel(value: LookingFor): string {
  return LOOKING_FOR.find((item) => item.value === value)?.label ?? "";
}

export function genderLabel(value: Gender): string {
  return GENDERS.find((item) => item.value === value)?.label.replace(/^(Une?|Un) /, "") ?? "";
}

export const REPORT_REASON_LABELS: Record<ReportReason, string> = {
  spam: "Spam ou arnaque",
  fake_profile: "Faux profil",
  harassment: "Harcèlement",
  inappropriate_content: "Contenu inapproprié",
  underage: "Personne mineure",
  other: "Autre"
};
