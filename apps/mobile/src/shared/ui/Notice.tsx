import React from "react";
import { StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

export type NoticeTone = "info" | "error" | "success";

/** Inline message banner; announced to screen readers when it changes. */
export function Notice({ message, tone = "info" }: { message: string; tone?: NoticeTone }) {
  const { colors } = useTheme();
  const palette = {
    info: { bg: colors.surfaceMuted, fg: colors.text, icon: "information-circle" as const },
    error: { bg: colors.dangerSoft, fg: colors.danger, icon: "alert-circle" as const },
    success: { bg: colors.successSoft, fg: colors.success, icon: "checkmark-circle" as const }
  }[tone];
  return (
    <View accessibilityLiveRegion="polite" accessibilityRole={tone === "error" ? "alert" : undefined} style={[styles.box, { backgroundColor: palette.bg }]}>
      <Ionicons color={palette.fg} name={palette.icon} size={20} />
      <Text style={[styles.text, { color: palette.fg }]} variant="bodySmall" weight="medium">
        {message}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  box: { flexDirection: "row", gap: spacing.sm, alignItems: "flex-start", borderRadius: radii.md, padding: spacing.md },
  text: { flex: 1 }
});
