import type { Preferences } from "../api/types";
import { DISTANCE_STEPS } from "./labels";

export const defaultPreferences: Preferences = {
  interestedIn: ["man", "woman", "non_binary"],
  minAge: 18,
  maxAge: 60,
  maxDistanceKm: 50,
  discoverable: true,
  showDistance: true
};

/** Next / previous distance in the preset ladder (null = "everywhere" is the last step). */
export function stepDistance(current: number | null, direction: -1 | 1): number | null {
  const index = DISTANCE_STEPS.findIndex((v) => v === current);
  const from = index === -1 ? DISTANCE_STEPS.indexOf(50) : index;
  const to = Math.min(DISTANCE_STEPS.length - 1, Math.max(0, from + direction));
  return DISTANCE_STEPS[to] ?? null;
}

export function toggleGender(list: Preferences["interestedIn"], gender: Preferences["interestedIn"][number]): Preferences["interestedIn"] {
  if (list.includes(gender)) return list.length > 1 ? list.filter((g) => g !== gender) : list; // at least one must stay selected
  return [...list, gender];
}
