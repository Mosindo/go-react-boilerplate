import React from "react";
import { View } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "../../ui";

/** The Lueur mark: a soft glow behind a serif wordmark. */
export function BrandMark({ size = "lg" }: { size?: "lg" | "sm" }) {
  const { colors } = useTheme();
  const glow = size === "lg" ? 120 : 64;
  return (
    <View style={{ alignItems: "center", gap: size === "lg" ? 18 : 8 }}>
      <View style={{ width: glow, height: glow, alignItems: "center", justifyContent: "center" }}>
        <View style={{ position: "absolute", width: glow, height: glow, borderRadius: glow / 2, backgroundColor: colors.primarySoft }} />
        <View style={{ position: "absolute", width: glow * 0.62, height: glow * 0.62, borderRadius: glow, backgroundColor: colors.accentSoft }} />
        <View style={{ width: glow * 0.26, height: glow * 0.26, borderRadius: glow, backgroundColor: colors.primary }} />
      </View>
      <Text accessibilityRole="header" variant={size === "lg" ? "display" : "title"}>
        Lueur
      </Text>
    </View>
  );
}
