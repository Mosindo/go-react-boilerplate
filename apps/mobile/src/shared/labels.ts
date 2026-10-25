import type { Gender, ReportReason } from "../api";

export const GENDERS: readonly Gender[] = ["man", "woman", "nonbinary"];

export const GENDER_LABELS: Record<Gender, string> = {
  man: "Homme",
  woman: "Femme",
  nonbinary: "Non-binaire"
};

export const INTERESTED_IN_LABELS: Record<Gender, string> = {
  man: "Des hommes",
  woman: "Des femmes",
  nonbinary: "Des personnes non-binaires"
};

export const REPORT_REASON_LABELS: Record<ReportReason, string> = {
  spam: "Spam ou arnaque",
  fake_profile: "Faux profil",
  harassment: "Harcèlement",
  inappropriate_content: "Contenu inapproprié",
  underage: "Personne mineure",
  other: "Autre"
};
