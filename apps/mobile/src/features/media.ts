/** Photo urls are relative ("/media/..."); prefix with the API base. Absolute urls pass through. */
export function resolveMediaUrl(baseUrl: string, url: string): string {
  if (/^https?:\/\//i.test(url)) {
    return url;
  }
  const base = baseUrl.replace(/\/+$/, "");
  return url.startsWith("/") ? `${base}${url}` : `${base}/${url}`;
}
