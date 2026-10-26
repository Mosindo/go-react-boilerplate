import React from "react";
import { Image, StyleSheet, View, type ImageStyle, type StyleProp } from "react-native";
import { API_BASE_URL } from "../api/client";
import { useTheme } from "../theme";
import { Text } from "./Text";

export function photoUri(url: string | null | undefined): string | undefined {
  return url ? `${API_BASE_URL}${url}` : undefined;
}

type PhotoProps = { url?: string | null; style?: StyleProp<ImageStyle>; label: string; testID?: string };

/** Remote photo with a neutral placeholder while missing. */
export function Photo({ url, style, label, testID }: PhotoProps) {
  const theme = useTheme();
  const uri = photoUri(url);
  if (!uri) {
    return (
      <View style={[{ backgroundColor: theme.surfaceAlt }, style as object]} accessibilityLabel={label} />
    );
  }
  return (
    <Image
      testID={testID}
      source={{ uri }}
      accessibilityLabel={label}
      resizeMode="cover"
      style={[{ backgroundColor: theme.surfaceAlt }, style]}
    />
  );
}

export function Avatar({ url, name, size = 48 }: { url?: string | null; name: string; size?: number }) {
  const theme = useTheme();
  const uri = photoUri(url);
  const box = { width: size, height: size, borderRadius: size / 2 };
  if (!uri) {
    return (
      <View style={[styles.fallback, box, { backgroundColor: theme.primarySoft }]} accessibilityLabel={name}>
        <Text variant="bodyStrong" color={theme.primary}>
          {name.slice(0, 1).toUpperCase()}
        </Text>
      </View>
    );
  }
  return (
    <Image
      source={{ uri }}
      accessibilityLabel={`Photo de ${name}`}
      style={[box, { backgroundColor: theme.surfaceAlt }]}
    />
  );
}

const styles = StyleSheet.create({
  fallback: { alignItems: "center", justifyContent: "center" }
});
