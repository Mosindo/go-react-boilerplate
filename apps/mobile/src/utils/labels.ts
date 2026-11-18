import type { Gender, RelationshipGoal, ReportReason } from "../api/types";

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

export const goalLabels: Record<RelationshipGoal, string> = {
  long_term: "Une relation durable",
  short_term: "Quelque chose de léger",
  friendship: "De nouvelles amitiés",
  unsure: "Je verrai bien"
};

export const reportReasonLabels: Record<ReportReason, string> = {
  fake_profile: "Faux profil ou usurpation",
  inappropriate_content: "Photos ou contenu inapproprié",
  harassment: "Harcèlement ou propos offensants",
  spam: "Spam, arnaque ou publicité",
  underage: "Personne mineure",
  other: "Autre raison"
};

export const genders: Gender[] = ["woman", "man", "nonbinary"];
export const goals: RelationshipGoal[] = ["long_term", "short_term", "friendship", "unsure"];
export const reportReasons: ReportReason[] = ["fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"];
export const distanceOptions = [5, 10, 25, 50, 100, 200, 500];
