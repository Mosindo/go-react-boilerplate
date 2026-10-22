import React, { useEffect, useState } from "react";
import { Image, Platform, StyleSheet, View, type ImageResizeMode, type StyleProp, type ViewStyle } from "react-native";
import { API_BASE_URL } from "../api/client";
import { endpoints } from "../api/endpoints";
import type { PhotoRef } from "../api/types";
import { useAuth } from "../hooks/useAuth";
import { Text, colors } from "../shared/ui";

type AuthImageProps = {
  photo?: PhotoRef | null;
  size?: "thumb" | "full";
  style?: StyleProp<ViewStyle>;
  resizeMode?: ImageResizeMode;
  label?: string;
  /** Shown while loading or when there is no photo. */
  placeholder?: string;
};

export function photoUri(photo: PhotoRef, size: "thumb" | "full"): string {
  const sizeParam = size === "thumb" ? "size=thumb&" : "";
  return `${API_BASE_URL}${endpoints.photoFile(photo.id)}?${sizeParam}v=${photo.version}`;
}

// Browsers cannot attach an Authorization header to <img>, so on the web photos are
// fetched with the token and displayed from an object URL (cached per image URL).
const blobCache = new Map<string, Promise<string>>();

function loadBlobUrl(uri: string, token: string): Promise<string> {
  let cached = blobCache.get(uri);
  if (!cached) {
    cached = fetch(uri, { headers: { Authorization: `Bearer ${token}` } }).then(async (response) => {
      if (!response.ok) {
        throw new Error(`photo ${response.status}`);
      }
      return URL.createObjectURL(await response.blob());
    });
    blobCache.set(uri, cached);
    cached.catch(() => blobCache.delete(uri));
  }
  return cached;
}

export function AuthImage({ photo, size = "full", style, resizeMode = "cover", label, placeholder }: AuthImageProps) {
  const { accessToken } = useAuth();
  const [webUri, setWebUri] = useState<string | null>(null);
  const uri = photo ? photoUri(photo, size) : null;

  useEffect(() => {
    if (Platform.OS !== "web" || !uri || !accessToken) {
      return undefined;
    }
    let cancelled = false;
    setWebUri(null);
    loadBlobUrl(uri, accessToken)
      .then((url) => {
        if (!cancelled) {
          setWebUri(url);
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [uri, accessToken]);

  const source =
    Platform.OS === "web"
      ? webUri
        ? { uri: webUri }
        : null
      : uri && accessToken
        ? { uri, headers: { Authorization: `Bearer ${accessToken}` } }
        : null;

  return (
    <View style={[styles.frame, style]}>
      {source ? (
        <Image accessibilityLabel={label} resizeMode={resizeMode} source={source} style={styles.image} />
      ) : placeholder ? (
        <Text style={styles.placeholder} tone="muted" variant="heading" weight="bold">
          {placeholder}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  frame: {
    backgroundColor: colors.surfaceSubtle,
    overflow: "hidden",
    alignItems: "center",
    justifyContent: "center"
  },
  image: { width: "100%", height: "100%" },
  placeholder: { textAlign: "center" }
});
