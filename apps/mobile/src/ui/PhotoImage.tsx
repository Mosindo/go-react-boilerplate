import React, { useEffect, useState } from "react";
import { Image, Platform, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { API_BASE_URL } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { useTheme } from "../theme";

type Props = {
  photoId: string | null | undefined;
  style?: StyleProp<ViewStyle>;
  resizeMode?: "cover" | "contain";
  accessibilityLabel?: string;
};

// Web <img> cannot send an Authorization header, so photos are fetched with the
// token and exposed as blob: URLs. Photo ids are immutable (replacing a photo
// issues a new id), so caching by id is safe.
const webCache = new Map<string, Promise<string>>();

function loadWebPhoto(photoId: string, token: string): Promise<string> {
  let hit = webCache.get(photoId);
  if (!hit) {
    hit = fetch(`${API_BASE_URL}/photos/${photoId}`, {
      headers: { Authorization: `Bearer ${token}` }
    })
      .then((res) => {
        if (!res.ok) throw new Error(`photo ${res.status}`);
        return res.blob();
      })
      .then((blob) => URL.createObjectURL(blob));
    webCache.set(photoId, hit);
    hit.catch(() => webCache.delete(photoId));
  }
  return hit;
}

function useWebPhotoUri(photoId: string | null | undefined, token: string | null): string | null {
  const [uri, setUri] = useState<string | null>(null);
  useEffect(() => {
    if (Platform.OS !== "web" || !photoId || !token) return;
    let cancelled = false;
    loadWebPhoto(photoId, token)
      .then((u) => !cancelled && setUri(u))
      .catch(() => !cancelled && setUri(null));
    return () => {
      cancelled = true;
    };
  }, [photoId, token]);
  return uri;
}

/** Photos are private: they are fetched with the bearer token, never from a public URL. */
export function PhotoImage({ photoId, style, resizeMode = "cover", accessibilityLabel }: Props) {
  const t = useTheme();
  const { accessToken } = useAuth();
  const [failed, setFailed] = useState(false);
  const webUri = useWebPhotoUri(photoId, accessToken);
  useEffect(() => setFailed(false), [photoId]);

  const source =
    Platform.OS === "web"
      ? webUri
        ? { uri: webUri }
        : null
      : photoId && accessToken
        ? {
            uri: `${API_BASE_URL}/photos/${photoId}`,
            headers: { Authorization: `Bearer ${accessToken}` }
          }
        : null;

  return (
    <View style={[styles.box, { backgroundColor: t.surfaceAlt }, style]}>
      {source && !failed ? (
        <Image
          accessibilityLabel={accessibilityLabel}
          accessible={!!accessibilityLabel}
          source={source}
          style={StyleSheet.absoluteFill}
          resizeMode={resizeMode}
          onError={() => setFailed(true)}
        />
      ) : (
        <View style={styles.placeholder}>
          <Ionicons name="person" size={36} color={t.textMuted} />
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  box: { overflow: "hidden" },
  placeholder: { flex: 1, alignItems: "center", justifyContent: "center" }
});
