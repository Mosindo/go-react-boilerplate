import type { Gender } from "../api/models";

export function formatAge(age: number | null | undefined): string {
  return typeof age === "number" && age > 0 ? String(age) : "";
}

export function formatNameAge(name: string, age: number | null | undefined): string {
  const label = formatAge(age);
  return label ? `${name}, ${label}` : name;
}

/** Distances from the API are already rounded server-side; we just phrase them. */
export function formatDistance(km: number | null | undefined): string {
  if (typeof km !== "number" || !Number.isFinite(km) || km < 0) {
    return "";
  }
  if (km <= 1) {
    return "Less than 1 km away";
  }
  return `${Math.round(km)} km away`;
}

export function formatKm(km: number): string {
  return `${Math.round(km)} km`;
}

export function formatAgeRange(min: number, max: number): string {
  return `${min} to ${max}`;
}

export function formatCount(count: number): string {
  if (count <= 0) {
    return "";
  }
  return count > 99 ? "99+" : String(count);
}

export const GENDER_LABELS: Record<Gender, string> = {
  woman: "Woman",
  man: "Man",
  non_binary: "Non-binary"
};

export const GENDER_INTEREST_LABELS: Record<Gender, string> = {
  woman: "Women",
  man: "Men",
  non_binary: "Non-binary people"
};

export function formatInterestedIn(genders: readonly Gender[]): string {
  if (genders.length === 0) {
    return "Nobody yet";
  }
  return genders.map((gender) => GENDER_INTEREST_LABELS[gender]).join(", ");
}

export function formatCounter(current: number, max: number): string {
  return `${current}/${max}`;
}
