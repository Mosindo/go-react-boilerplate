import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { radii, spacing } from "./tokens";

type NoticeProps = {
  message: string;
  tone?: "info" | "success" | "warning" | "danger";
  style?: StyleProp<ViewStyle>;
};

export function Notice({ message, tone = "info", style }: NoticeProps) {
  const { colors } = useTheme();
  const palette = {
    info: { bg: colors.surfaceAccent, border: colors.primaryBorder, text: colors.text },
    success: { bg: colors.successSoft, border: colors.successBorder, text: colors.success },
    warning: { bg: colors.warningSoft, border: colors.warningBorder, text: colors.warning },
    danger: { bg: colors.dangerSoft, border: colors.dangerBorder, text: colors.danger }
  }[tone];
  return (
    <View
      accessible
      accessibilityLiveRegion="polite"
      accessibilityRole={tone === "danger" ? "alert" : undefined}
      style={[styles.box, { backgroundColor: palette.bg, borderColor: palette.border }, style]}
    >
      <Text variant="label" style={{ color: palette.text }}>
        {message}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  box: { borderWidth: 1, borderRadius: radii.md, paddingHorizontal: spacing.md, paddingVertical: spacing.sm }
});
