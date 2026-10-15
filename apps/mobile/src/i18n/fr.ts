import type { Gender, ReportReason } from "../api/types";

export const APP_NAME = "Alma";

export const genderLabels: Record<Gender, string> = {
  man: "Un homme",
  woman: "Une femme",
  nonbinary: "Non-binaire"
};

export const genderShort: Record<Gender, string> = {
  man: "Homme",
  woman: "Femme",
  nonbinary: "Non-binaire"
};

export const interestedInLabels: Record<Gender, string> = {
  man: "Des hommes",
  woman: "Des femmes",
  nonbinary: "Des personnes non-binaires"
};

export const reportReasons: { value: ReportReason; label: string; hint: string }[] = [
  { value: "harassment", label: "Harcèlement ou comportement agressif", hint: "Messages insultants, pressions, menaces" },
  { value: "fake", label: "Faux profil ou usurpation", hint: "Photos ou informations qui semblent fausses" },
  { value: "inappropriate", label: "Contenu inapproprié", hint: "Photos ou propos choquants" },
  { value: "spam", label: "Spam ou arnaque", hint: "Publicité, demande d'argent, liens suspects" },
  { value: "underage", label: "Personne qui semble mineure", hint: "Alma est réservée aux 18 ans et plus" },
  { value: "other", label: "Autre raison", hint: "Dites-nous en plus" }
];

export const distanceChoices = [10, 25, 50, 100, 200, 500];
