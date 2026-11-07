export function formatDistance(km: number | null | undefined): string | null {
  if (km === null || km === undefined) return null;
  return km <= 5 ? "à moins de 5 km" : `à ${km} km`;
}

export function formatTime(iso: string, now: Date = new Date()): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const sameDay = d.toDateString() === now.toDateString();
  if (sameDay) return d.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
  const diffDays = (now.getTime() - d.getTime()) / 86_400_000;
  if (diffDays < 7) return d.toLocaleDateString("fr-FR", { weekday: "short" });
  return d.toLocaleDateString("fr-FR", { day: "2-digit", month: "short" });
}

/** True when the message was read by the other person (they opened the chat after it was sent). */
export function isReadByPeer(createdAt: string, peerReadAt: string | null): boolean {
  if (!peerReadAt) return false;
  return new Date(peerReadAt).getTime() >= new Date(createdAt).getTime();
}
