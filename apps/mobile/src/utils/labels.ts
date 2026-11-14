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
  casual: "Quelque chose de léger",
  friendship: "De nouvelles amitiés",
  not_sure: "Je verrai bien"
};

export const reportReasonLabels: Record<ReportReason, string> = {
  fake_profile: "Faux profil ou usurpation",
  inappropriate_content: "Photos ou contenu inapproprié",
  harassment: "Harcèlement ou propos offensants",
  spam: "Arnaque, spam ou publicité",
  underage: "Personne mineure",
  other: "Autre chose"
};

export const genders: Gender[] = ["woman", "man", "nonbinary"];
export const lookingFors: LookingFor[] = ["relationship", "casual", "friendship", "not_sure"];
export const reportReasons: ReportReason[] = ["fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"];
