import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { radii, spacing } from "./tokens";

export type NoticeProps = {
  message: string;
  tone?: "info" | "success" | "warning" | "danger";
  style?: StyleProp<ViewStyle>;
};

export function Notice({ message, tone = "info", style }: NoticeProps) {
  const { colors } = useTheme();
  const palette = {
    info: { bg: colors.primarySoft, border: colors.primaryBorder, fg: colors.text },
    success: { bg: colors.successSoft, border: colors.successBorder, fg: colors.success },
    warning: { bg: colors.warningSoft, border: colors.warningBorder, fg: colors.warning },
    danger: { bg: colors.dangerSoft, border: colors.dangerBorder, fg: colors.danger }
  }[tone];
  return (
    <View
      accessibilityLiveRegion="polite"
      accessibilityRole={tone === "danger" ? "alert" : undefined}
      style={[styles.root, { backgroundColor: palette.bg, borderColor: palette.border }, style]}
    >
      <Text style={{ color: palette.fg }} variant="label">
        {message}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { borderWidth: 1, borderRadius: radii.md, padding: spacing.md }
});
