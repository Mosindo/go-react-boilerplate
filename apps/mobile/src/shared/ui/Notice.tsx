import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../../theme/useTheme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

export type NoticeTone = "default" | "danger" | "success" | "warning";

export type NoticeProps = {
  description?: string;
  style?: StyleProp<ViewStyle>;
  title: string;
  tone?: NoticeTone;
};

const titleToneByNotice: Record<NoticeTone, "danger" | "default" | "secondary" | "success"> = {
  default: "default",
  danger: "danger",
  success: "success",
  warning: "secondary"
};

export function Notice({ description, style, title, tone = "default" }: NoticeProps) {
  const { colors } = useTheme();
  const palette: Record<NoticeTone, ViewStyle> = {
    default: { backgroundColor: colors.surfaceSubtle, borderColor: colors.border },
    danger: { backgroundColor: colors.dangerSoft, borderColor: colors.dangerBorder },
    success: { backgroundColor: colors.successSoft, borderColor: colors.successBorder },
    warning: { backgroundColor: colors.warningSoft, borderColor: colors.warningBorder }
  };

  return (
    <View
      style={[
        { borderRadius: radii.lg, borderWidth: 1, padding: spacing.md, gap: spacing.xs },
        palette[tone],
        style
      ]}
    >
      <Text tone={titleToneByNotice[tone]} variant="label" weight="bold">
        {title}
      </Text>
      {description ? <Text tone="secondary">{description}</Text> : null}
    </View>
  );
}
