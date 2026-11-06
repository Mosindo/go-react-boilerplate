import React from "react";
import { StyleSheet, View } from "react-native";
import { initials } from "../lib/format";
import { useTheme } from "../theme";
import { AuthImage } from "./AuthImage";
import { Text } from "./ui";

export function Avatar({
  name,
  photoId,
  size = 52
}: {
  name: string;
  photoId?: string | null;
  size?: number;
}) {
  const { colors } = useTheme();
  const box = { width: size, height: size, borderRadius: size / 2 };
  if (photoId) {
    return (
      <View style={[box, { overflow: "hidden", backgroundColor: colors.surfaceAlt }]}>
        <AuthImage
          photoId={photoId}
          style={StyleSheet.absoluteFill as object}
          accessibilityLabel={`Photo de ${name}`}
        />
      </View>
    );
  }
  return (
    <View
      style={[
        box,
        { backgroundColor: colors.primarySoft, alignItems: "center", justifyContent: "center" }
      ]}
    >
      <Text variant="heading" color={colors.primary}>
        {initials(name)}
      </Text>
    </View>
  );
}
