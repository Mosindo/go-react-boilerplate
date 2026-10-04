import React, { useEffect, useState, useSyncExternalStore } from "react";
import { Image, Platform, StyleSheet, View, type ImageStyle, type StyleProp, type ViewStyle } from "react-native";
import { API_BASE_URL } from "../api/config";
import type { Photo } from "../api/types";
import { initialOf } from "../lib/format";
import { tokenStore } from "../store/tokenStore";
import { useColors } from "../theme";
import { Text } from "./Text";

/** Current access token as a hook, so photo components re-render after a token refresh. */
export function useAccessToken(): string | null {
  return useSyncExternalStore(tokenStore.subscribe, tokenStore.getAccessToken, tokenStore.getAccessToken);
}

/** Pure: builds the authenticated image source for a photo. */
export function photoSource(photo: Photo, size: "thumb" | "full", token: string | null, baseUrl = API_BASE_URL) {
  const path = size === "thumb" ? photo.thumbUrl : photo.url;
  return {
    uri: `${baseUrl}${path}`,
    headers: token ? { Authorization: `Bearer ${token}` } : undefined
  };
}

const webBlobCache = new Map<string, string>();

/**
 * Browsers cannot attach an Authorization header to <img>, so on web the photo is fetched with the
 * token and shown through an object URL (cached per URL, photos are immutable). Native passes headers directly.
 */
function useWebBlobUri(uri: string | null, token: string | null): { uri: string | null; failed: boolean } {
  const enabled = Platform.OS === "web" && uri !== null && token !== null;
  const [state, setState] = useState<{ key: string; uri: string | null; failed: boolean }>({
    key: "",
    uri: null,
    failed: false
  });
  useEffect(() => {
    if (!enabled || !uri) return;
    const cached = webBlobCache.get(uri);
    if (cached) {
      setState({ key: uri, uri: cached, failed: false });
      return;
    }
    let cancelled = false;
    fetch(uri, { headers: { Authorization: `Bearer ${token}` } })
      .then((res) => (res.ok ? res.blob() : Promise.reject(new Error(String(res.status)))))
      .then((blob) => {
        const url = URL.createObjectURL(blob);
        webBlobCache.set(uri, url);
        if (!cancelled) setState({ key: uri, uri: url, failed: false });
      })
      .catch(() => {
        if (!cancelled) setState({ key: uri, uri: null, failed: true });
      });
    return () => {
      cancelled = true;
    };
  }, [enabled, uri, token]);
  if (!enabled) return { uri: null, failed: false };
  return state.key === uri
    ? { uri: state.uri, failed: state.failed }
    : { uri: webBlobCache.get(uri!) ?? null, failed: false };
}

type ProfilePhotoProps = {
  photo: Photo | null | undefined;
  /** thumb for lists/avatars, full for cards and detail. */
  size?: "thumb" | "full";
  /** Used for the placeholder initial and the accessibility label. */
  name?: string;
  style?: StyleProp<ImageStyle>;
  round?: boolean;
};

export function ProfilePhoto({ photo, size = "thumb", name = "", style, round }: ProfilePhotoProps) {
  const token = useAccessToken();
  const colors = useColors();
  const [failed, setFailed] = useState(false);
  const photoId = photo?.id;
  useEffect(() => setFailed(false), [photoId]);

  const source = photo && token ? photoSource(photo, size, token) : null;
  const web = useWebBlobUri(source?.uri ?? null, token);
  const isWeb = Platform.OS === "web";

  const flat = StyleSheet.flatten(style) ?? {};
  const radiusStyle = round ? { borderRadius: 9999 } : null;

  if (!photo || !token || failed || (isWeb && web.failed)) {
    return (
      <View
        accessibilityLabel={name ? `${name}, no photo` : "No photo"}
        style={[styles.placeholder, { backgroundColor: colors.primarySoft }, flat as StyleProp<ViewStyle>, radiusStyle]}
      >
        <Text style={{ color: colors.primary }} variant="title" weight="bold">
          {initialOf(name)}
        </Text>
      </View>
    );
  }

  if (isWeb && !web.uri) {
    return <View style={[{ backgroundColor: colors.skeleton }, flat as StyleProp<ViewStyle>, radiusStyle]} />;
  }

  return (
    <Image
      accessibilityIgnoresInvertColors
      accessibilityLabel={name ? `Photo of ${name}` : "Profile photo"}
      onError={() => setFailed(true)}
      resizeMode="cover"
      source={isWeb && web.uri ? { uri: web.uri } : photoSource(photo, size, token)}
      style={[{ backgroundColor: colors.skeleton }, flat, radiusStyle]}
    />
  );
}

const styles = StyleSheet.create({
  placeholder: { alignItems: "center", justifyContent: "center", overflow: "hidden" }
});
