import React, { useEffect, useMemo, useState } from "react";
import { Platform, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Image, type ImageStyle } from "expo-image";
import { assetUrl } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { Text, colors } from "../shared/ui";

const blobCache = new Map<string, string>();

/**
 * Browsers cannot attach an Authorization header to <img>, so on web the bytes are fetched with
 * the token and shown through an object URL. Native uses expo-image headers instead.
 */
function useBlobUrl(path: string | null | undefined, token: string | null, enabled: boolean): string | null {
  const [url, setUrl] = useState<string | null>(path ? (blobCache.get(path) ?? null) : null);
  useEffect(() => {
    if (!enabled || !path || !token) {
      return;
    }
    const cached = blobCache.get(path);
    if (cached) {
      setUrl(cached);
      return;
    }
    let cancelled = false;
    fetch(assetUrl(path), { headers: { Authorization: `Bearer ${token}` } })
      .then((response) => (response.ok ? response.blob() : Promise.reject(new Error(String(response.status)))))
      .then((blob) => {
        const objectUrl = URL.createObjectURL(blob);
        blobCache.set(path, objectUrl);
        if (!cancelled) {
          setUrl(objectUrl);
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [enabled, path, token]);
  return url;
}

type PhotoImageProps = {
  path?: string | null;
  style?: StyleProp<ImageStyle>;
  fallbackLabel?: string;
  accessibilityLabel?: string;
  contentFit?: "cover" | "contain";
};

/** Photos are private: every request carries the bearer token. expo-image caches them on disk. */
export function PhotoImage({ accessibilityLabel, contentFit = "cover", fallbackLabel, path, style }: PhotoImageProps) {
  const { accessToken } = useAuth();
  const isWeb = Platform.OS === "web";
  const blobUrl = useBlobUrl(path, accessToken, isWeb);
  const source = useMemo(() => {
    if (isWeb) {
      return blobUrl ? { uri: blobUrl } : null;
    }
    return path && accessToken ? { uri: assetUrl(path), headers: { Authorization: `Bearer ${accessToken}` }, cacheKey: path } : null;
  }, [accessToken, blobUrl, isWeb, path]);

  if (!source) {
    return (
      <View style={[styles.fallback, style as StyleProp<ViewStyle>]}>
        <Text tone="muted" weight="bold" variant="heading">
          {(fallbackLabel ?? "?").slice(0, 1).toUpperCase()}
        </Text>
      </View>
    );
  }
  return (
    <Image
      accessibilityLabel={accessibilityLabel}
      accessible={Boolean(accessibilityLabel)}
      cachePolicy="disk"
      contentFit={contentFit}
      source={source}
      style={[styles.image, style]}
      transition={150}
    />
  );
}

const styles = StyleSheet.create({
  image: { backgroundColor: colors.surfaceSubtle },
  fallback: { alignItems: "center", justifyContent: "center", backgroundColor: colors.surfaceSubtle }
});
