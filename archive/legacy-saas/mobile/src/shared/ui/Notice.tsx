import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { colors, radii, spacing } from "./tokens";

export type NoticeTone = "default" | "danger" | "success" | "warning";

export type NoticeProps = {
  description?: string;
  style?: StyleProp<ViewStyle>;
  title: string;
  tone?: NoticeTone;
};

const styles = StyleSheet.create({
  base: {
    borderRadius: radii.lg,
    borderWidth: 1,
    padding: spacing.md,
    gap: spacing.xs
  },
  default: {
    backgroundColor: colors.surfaceSubtle,
    borderColor: colors.border
  },
  danger: {
    backgroundColor: colors.dangerSoft,
    borderColor: colors.dangerBorder
  },
  success: {
    backgroundColor: colors.successSoft,
    borderColor: colors.successBorder
  },
  warning: {
    backgroundColor: colors.warningSoft,
    borderColor: colors.warningBorder
  }
});

const toneStyles: Record<NoticeTone, ViewStyle> = {
  default: styles.default,
  danger: styles.danger,
  success: styles.success,
  warning: styles.warning
};

const titleToneByNotice: Record<NoticeTone, "danger" | "default" | "secondary" | "success"> = {
  default: "default",
  danger: "danger",
  success: "success",
  warning: "secondary"
};

export function Notice({ description, style, title, tone = "default" }: NoticeProps) {
  return (
    <View style={[styles.base, toneStyles[tone], style]}>
      <Text tone={titleToneByNotice[tone]} variant="label" weight="bold">
        {title}
      </Text>
      {description ? <Text tone="secondary">{description}</Text> : null}
    </View>
  );
}
