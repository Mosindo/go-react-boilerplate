import React, { useEffect, useState } from "react";
import { Image as NativeImage, Platform, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Image as ExpoImage } from "expo-image";
import { absoluteUrl, apiBlob, authHeaders } from "../../api/client";
import { useTheme } from "../../theme";

const blobCache = new Map<string, string>();
const MAX_CACHED = 80;

/** Web only: <img> cannot send the bearer token, so photos are fetched and exposed as blob URLs. */
function useWebBlobUrl(path: string | undefined, enabled: boolean): string | null {
  const [url, setUrl] = useState<string | null>(path ? (blobCache.get(path) ?? null) : null);
  useEffect(() => {
    if (!enabled || !path) return;
    const cached = blobCache.get(path);
    if (cached) {
      setUrl(cached);
      return;
    }
    let active = true;
    apiBlob(path)
      .then((blob) => {
        const objectUrl = URL.createObjectURL(blob);
        if (blobCache.size >= MAX_CACHED) {
          const oldest = blobCache.keys().next().value;
          if (oldest !== undefined) {
            URL.revokeObjectURL(blobCache.get(oldest) ?? "");
            blobCache.delete(oldest);
          }
        }
        blobCache.set(path, objectUrl);
        if (active) setUrl(objectUrl);
      })
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, [path, enabled]);
  return url;
}

export type PhotoImageProps = {
  /** API-relative path such as `/photos/<id>` or `/photos/<id>?size=thumb`. */
  path?: string;
  style?: StyleProp<ViewStyle>;
  fit?: "cover" | "contain";
  label?: string;
};

/** Authenticated image with a neutral placeholder while loading or when absent. */
export function PhotoImage({ path, style, fit = "cover", label }: PhotoImageProps) {
  const { colors } = useTheme();
  const isWeb = Platform.OS === "web";
  const blobUrl = useWebBlobUrl(path, isWeb);
  const placeholder = <View style={[StyleSheet.absoluteFill, { backgroundColor: colors.surfaceMuted }]} />;

  return (
    <View accessibilityLabel={label} accessibilityRole="image" style={[styles.box, { backgroundColor: colors.surfaceMuted }, style]}>
      {!path ? (
        placeholder
      ) : isWeb ? (
        blobUrl ? (
          <NativeImage resizeMode={fit} source={{ uri: blobUrl }} style={StyleSheet.absoluteFill} />
        ) : (
          placeholder
        )
      ) : (
        <ExpoImage
          cachePolicy="disk"
          contentFit={fit}
          recyclingKey={path}
          source={{ uri: absoluteUrl(path), headers: authHeaders() }}
          style={StyleSheet.absoluteFill}
          transition={150}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  box: { overflow: "hidden" }
});
