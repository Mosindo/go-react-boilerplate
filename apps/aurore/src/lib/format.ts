import type { Card } from "../api/types";

export function distanceLabel(km: number | null): string | null {
  if (km === null) return null;
  return km <= 5 ? "à moins de 5 km" : `à environ ${km} km`;
}

export function genderLabel(g: string): string {
  switch (g) {
    case "man":
      return "Homme";
    case "woman":
      return "Femme";
    case "non_binary":
      return "Non-binaire";
    default:
      return "";
  }
}

export function genderPlural(g: string): string {
  switch (g) {
    case "man":
      return "Des hommes";
    case "woman":
      return "Des femmes";
    case "non_binary":
      return "Des personnes non-binaires";
    default:
      return "";
  }
}

export function headline(card: Pick<Card, "firstName" | "age">): string {
  return `${card.firstName}, ${card.age}`;
}

export function truncate(text: string, max: number): string {
  return text.length <= max ? text : `${text.slice(0, max - 1).trimEnd()}…`;
}

export const REPORT_REASONS = [
  { value: "spam", label: "Spam ou publicité" },
  { value: "fake", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
] as const;
