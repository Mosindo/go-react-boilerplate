import { useMemo } from "react";
import { API_BASE_URL } from "../api/client";
import { useAuth } from "../hooks/useAuth";

export type AuthedImageSource = { uri: string; headers: { Authorization: string } };

/** Photos are private: images are fetched with the bearer token, never from a public URL. */
export function useAuthedImage(path: string | null | undefined): AuthedImageSource | null {
  const { accessToken } = useAuth();
  return useMemo(() => {
    if (!path || !accessToken || !API_BASE_URL) {
      return null;
    }
    return { uri: `${API_BASE_URL}${path}`, headers: { Authorization: `Bearer ${accessToken}` } };
  }, [path, accessToken]);
}
