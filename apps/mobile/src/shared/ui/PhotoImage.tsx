import React, { useEffect, useState } from "react";
import { Image, Platform, StyleSheet, View, type ImageStyle, type StyleProp } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { API_BASE_URL } from "../../api/client";
import { endpoints } from "../../api/endpoints";
import { peekTokens } from "../../store/tokenStore";
import { useTheme } from "../../theme";

type PhotoImageProps = {
  photoId: string | null | undefined;
  size?: "thumb" | "full";
  style?: StyleProp<ImageStyle>;
  accessibilityLabel?: string;
};

// Web <img> cannot send an Authorization header, so photos are fetched with the
// token and exposed as object URLs (cached per photo id: ids are immutable).
const webCache = new Map<string, Promise<string>>();

function loadWebPhoto(url: string, token: string): Promise<string> {
  let pending = webCache.get(url);
  if (!pending) {
    pending = fetch(url, { headers: { Authorization: `Bearer ${token}` } }).then(async (response) => {
      if (!response.ok) {
        throw new Error(`photo ${response.status}`);
      }
      return URL.createObjectURL(await response.blob());
    });
    pending.catch(() => webCache.delete(url));
    webCache.set(url, pending);
  }
  return pending;
}

/** Image for a protected photo; falls back to a neutral placeholder. */
export function PhotoImage({ photoId, size = "full", style, accessibilityLabel }: PhotoImageProps) {
  const { colors } = useTheme();
  const [webUri, setWebUri] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);
  const token = peekTokens()?.accessToken ?? null;
  const url = photoId ? `${API_BASE_URL}${endpoints.photos.file(photoId, size)}` : null;

  useEffect(() => {
    setFailed(false);
    if (Platform.OS !== "web" || !url || !token) {
      return;
    }
    let active = true;
    loadWebPhoto(url, token)
      .then((objectUrl) => active && setWebUri(objectUrl))
      .catch(() => active && setFailed(true));
    return () => {
      active = false;
    };
  }, [url, token]);

  const source =
    !url || !token || failed
      ? null
      : Platform.OS === "web"
        ? webUri
          ? { uri: webUri }
          : null
        : { uri: url, headers: { Authorization: `Bearer ${token}` } };

  if (!source) {
    return (
      <View accessibilityLabel={accessibilityLabel} style={[styles.placeholder, { backgroundColor: colors.surfaceMuted }, style as object]}>
        <Ionicons color={colors.textSubtle} name="person" size={28} />
      </View>
    );
  }
  return (
    <Image
      accessibilityLabel={accessibilityLabel}
      onError={() => setFailed(true)}
      resizeMode="cover"
      source={source}
      style={[{ backgroundColor: colors.surfaceMuted }, style]}
    />
  );
}

const styles = StyleSheet.create({
  placeholder: { alignItems: "center", justifyContent: "center" }
});
