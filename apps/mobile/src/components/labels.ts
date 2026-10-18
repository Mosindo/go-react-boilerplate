import type { Gender } from "../api/types";

export const APP_NAME = "Tendre";

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

export const GENDERS: Gender[] = ["man", "woman", "nonbinary"];

export const DISTANCE_OPTIONS: { label: string; value: number | null }[] = [
  { label: "10 km", value: 10 },
  { label: "25 km", value: 25 },
  { label: "50 km", value: 50 },
  { label: "100 km", value: 100 },
  { label: "250 km", value: 250 },
  { label: "Partout", value: null }
];

export const MIN_AGE = 18;
export const MAX_AGE = 99;
export const MAX_BIO = 500;
export const MAX_INTERESTS = 8;
export const MAX_PHOTOS = 6;

export function formatDistance(km: number | null): string | null {
  if (km === null) {
    return null;
  }
  return km <= 1 ? "À moins de 2 km" : `À environ ${km} km`;
}
