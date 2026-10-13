import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

type NoticeProps = {
  message: string;
  title?: string;
  tone?: "info" | "success" | "warning" | "danger";
  style?: StyleProp<ViewStyle>;
};

/** Inline banner for form-level feedback. */
export function Notice({ message, title, tone = "info", style }: NoticeProps) {
  const { colors } = useTheme();
  const palette = {
    info: { bg: colors.primarySoft, fg: colors.primary },
    success: { bg: colors.successSoft, fg: colors.success },
    warning: { bg: colors.warningSoft, fg: colors.warning },
    danger: { bg: colors.dangerSoft, fg: colors.danger }
  }[tone];

  return (
    <View
      accessibilityLiveRegion="polite"
      accessibilityRole={tone === "danger" ? "alert" : undefined}
      style={[styles.root, { backgroundColor: palette.bg }, style]}
    >
      {title ? (
        <Text style={{ color: palette.fg }} variant="label" weight="bold">
          {title}
        </Text>
      ) : null}
      <Text style={{ color: palette.fg }} variant="label">
        {message}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { borderRadius: radii.md, padding: spacing.md, gap: spacing.xxs }
});
