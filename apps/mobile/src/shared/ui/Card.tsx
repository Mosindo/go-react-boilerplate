import React from "react";
import { StyleSheet, View, type ViewProps } from "react-native";
import { colors, radii, spacing } from "./tokens";

type CardProps = ViewProps & {
  padded?: boolean;
  tone?: "default" | "raised" | "primary";
};

export function Card({ padded = true, style, tone = "default", ...props }: CardProps) {
  return (
    <View
      style={[
        styles.base,
        tone === "raised" ? styles.raised : null,
        tone === "primary" ? styles.primary : null,
        padded ? styles.padded : null,
        style
      ]}
      {...props}
    />
  );
}

const styles = StyleSheet.create({
  base: {
    backgroundColor: colors.surface,
    borderRadius: radii.lg,
    borderWidth: 1,
    borderColor: colors.border
  },
  raised: {
    backgroundColor: colors.surfaceRaised
  },
  primary: {
    backgroundColor: colors.primarySoft,
    borderColor: colors.primaryBorder
  },
  padded: {
    padding: spacing.lg,
    gap: spacing.md
  }
});
