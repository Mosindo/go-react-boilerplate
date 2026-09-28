/** Distances are bucketed to 5 km steps by the server (minimum 5). */
export function formatDistance(distanceKm: number | null | undefined): string | null {
  if (distanceKm === null || distanceKm === undefined || !Number.isFinite(distanceKm)) {
    return null;
  }
  if (distanceKm <= 5) {
    return "Within 5 km";
  }
  return `About ${Math.round(distanceKm)} km away`;
}
