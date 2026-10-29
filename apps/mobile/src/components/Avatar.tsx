import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { fontSizes } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";
import { AuthImage } from "./AuthImage";

type Props = { name: string; url?: string | null; size?: number };

export function Avatar({ name, url, size = 52 }: Props) {
  const theme = useTheme();
  const radius = size / 2;
  if (url) {
    return (
      <AuthImage
        accessibilityLabel={`Photo de ${name}`}
        style={{ width: size, height: size, borderRadius: radius }}
        url={url}
      />
    );
  }
  return (
    <View
      accessibilityLabel={name}
      style={[
        styles.fallback,
        { width: size, height: size, borderRadius: radius, backgroundColor: theme.primarySoft }
      ]}
    >
      <Text style={[styles.initial, { color: theme.primary, fontSize: size * 0.4 }]}>
        {name.trim().charAt(0).toUpperCase()}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  fallback: { alignItems: "center", justifyContent: "center" },
  initial: { fontWeight: "700", fontSize: fontSizes.lg }
});
