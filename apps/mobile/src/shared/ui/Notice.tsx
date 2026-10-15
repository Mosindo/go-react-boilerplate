import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

export type NoticeTone = "info" | "success" | "warning" | "danger";

export type NoticeProps = {
  message: string;
  title?: string;
  tone?: NoticeTone;
  style?: StyleProp<ViewStyle>;
};

export function Notice({ message, style, title, tone = "info" }: NoticeProps) {
  const palette = useTheme();
  const tones: Record<NoticeTone, { bg: string; fg: string }> = {
    info: { bg: palette.brandSoft, fg: palette.brand },
    success: { bg: palette.successSoft, fg: palette.success },
    warning: { bg: palette.warningSoft, fg: palette.warning },
    danger: { bg: palette.dangerSoft, fg: palette.danger }
  };
  const t = tones[tone];
  return (
    <View
      accessibilityLiveRegion="polite"
      accessibilityRole={tone === "danger" ? "alert" : undefined}
      style={[{ backgroundColor: t.bg, borderRadius: radii.md, padding: spacing.md, gap: spacing.xxs }, style]}
    >
      {title ? (
        <Text style={{ color: t.fg }} weight="semibold">
          {title}
        </Text>
      ) : null}
      <Text style={{ color: t.fg }} variant="label">
        {message}
      </Text>
    </View>
  );
}
