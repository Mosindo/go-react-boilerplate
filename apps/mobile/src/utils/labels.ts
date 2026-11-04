import type { Gender, ReportReason } from "../api/types";

export const GENDERS: { value: Gender; label: string; plural: string }[] = [
  { value: "woman", label: "Femme", plural: "Femmes" },
  { value: "man", label: "Homme", plural: "Hommes" },
  { value: "nonbinary", label: "Non-binaire", plural: "Personnes non-binaires" },
];

export const REPORT_REASONS: { value: ReportReason; label: string }[] = [
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "harassment", label: "Harcèlement ou comportement abusif" },
  { value: "fake", label: "Faux profil" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" },
];
