/** "~5 km away"; null when unknown or hidden by the other user. */
export function formatDistance(distanceKm: number | null | undefined): string | null {
  if (distanceKm === null || distanceKm === undefined || !Number.isFinite(distanceKm) || distanceKm < 0) {
    return null;
  }
  if (distanceKm < 1) {
    return "Less than 1 km away";
  }
  return `~${Math.round(distanceKm)} km away`;
}

export function sharedInterestSet(slugs: readonly string[]): ReadonlySet<string> {
  return new Set(slugs);
}
