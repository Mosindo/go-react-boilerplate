import React, { useEffect, useState } from "react";
import { Image, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { API_BASE_URL, getTokens, subscribeTokens } from "../api/client";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type Props = {
  photoId?: string | null;
  name?: string;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

/** Photos are private: every fetch carries the bearer token. */
export function photoSource(photoId: string) {
  const token = getTokens()?.accessToken;
  return {
    uri: `${API_BASE_URL}/photos/${photoId}`,
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    cache: "force-cache" as const
  };
}

export function PhotoImage({ photoId, name, style, testID }: Props) {
  const { colors } = useTheme();
  const [, rerender] = useState(0);
  const [failedId, setFailedId] = useState<string | null>(null);
  const failed = !!photoId && failedId === photoId;

  // Re-render after a token refresh so retried loads use the new credentials.
  useEffect(() => subscribeTokens(() => rerender((n) => n + 1)), []);

  if (!photoId || failed) {
    return (
      <View style={[styles.fallback, { backgroundColor: colors.accentSoft }, style]} testID={testID}>
        <Text variant="title" tone="accent">
          {(name ?? "?").slice(0, 1).toUpperCase()}
        </Text>
      </View>
    );
  }
  return (
    <Image
      testID={testID}
      source={photoSource(photoId)}
      style={[{ backgroundColor: colors.surfaceAlt }, style as object]}
      resizeMode="cover"
      onError={() => setFailedId(photoId ?? null)}
      accessibilityIgnoresInvertColors
      accessibilityLabel={name ? `Photo de ${name}` : "Photo"}
    />
  );
}

export function Avatar({
  photoId,
  name,
  size = 56
}: {
  photoId?: string | null;
  name?: string;
  size?: number;
}) {
  return (
    <PhotoImage
      photoId={photoId}
      name={name}
      style={{ width: size, height: size, borderRadius: size / 2, overflow: "hidden" }}
    />
  );
}

const styles = StyleSheet.create({ fallback: { alignItems: "center", justifyContent: "center" } });
