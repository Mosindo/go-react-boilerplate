import React from "react";
import { View } from "react-native";
import { useTheme } from "../theme";

/** Lueur's mark: a soft glow (three concentric discs). */
export function BrandMark({ size = 72 }: { size?: number }) {
  const { colors } = useTheme();
  return (
    <View
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
      style={{
        width: size,
        height: size,
        borderRadius: size / 2,
        backgroundColor: colors.primarySoft,
        alignItems: "center",
        justifyContent: "center"
      }}
    >
      <View
        style={{
          width: size * 0.62,
          height: size * 0.62,
          borderRadius: size,
          backgroundColor: colors.primary,
          opacity: 0.35,
          alignItems: "center",
          justifyContent: "center"
        }}
      >
        <View style={{ width: size * 0.32, height: size * 0.32, borderRadius: size, backgroundColor: colors.primary }} />
      </View>
    </View>
  );
}
