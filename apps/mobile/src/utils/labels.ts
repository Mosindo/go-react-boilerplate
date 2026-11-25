import type { Gender, LookingFor, ReportReason } from "../api/types";

export const genderLabels: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire"
};

export const interestedInLabels: Record<Gender, string> = {
  woman: "Des femmes",
  man: "Des hommes",
  nonbinary: "Des personnes non-binaires"
};

export const lookingForLabels: Record<LookingFor, string> = {
  relationship: "Une relation sérieuse",
  casual: "Des rencontres légères",
  friendship: "De nouvelles amitiés",
  unsure: "Je verrai bien"
};

export const reportReasonLabels: Record<ReportReason, string> = {
  fake_profile: "Faux profil",
  inappropriate_content: "Contenu inapproprié",
  harassment: "Harcèlement ou propos offensants",
  spam: "Spam ou arnaque",
  underage: "Personne mineure",
  other: "Autre raison"
};

export const genders: Gender[] = ["woman", "man", "nonbinary"];
export const lookingForOptions: LookingFor[] = ["relationship", "casual", "friendship", "unsure"];
export const reportReasons: ReportReason[] = ["fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"];
