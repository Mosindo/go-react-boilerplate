import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Text, type TextTone } from "./Text";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type NoticeTone = "default" | "danger" | "success" | "warning";

export type NoticeProps = {
  description?: string;
  style?: StyleProp<ViewStyle>;
  title: string;
  tone?: NoticeTone;
  testID?: string;
};

const makeStyles = (t: Theme) => ({
  base: { borderRadius: t.radii.md, borderWidth: 1, padding: t.spacing.md, gap: t.spacing.xxs },
  default: { backgroundColor: t.colors.surfaceMuted, borderColor: t.colors.border },
  danger: { backgroundColor: t.colors.dangerSoft, borderColor: t.colors.danger },
  success: { backgroundColor: t.colors.successSoft, borderColor: t.colors.success },
  warning: { backgroundColor: t.colors.warningSoft, borderColor: t.colors.warning }
});

const titleToneByNotice: Record<NoticeTone, TextTone> = {
  default: "default",
  danger: "danger",
  success: "success",
  warning: "default"
};

export function Notice({ description, style, testID, title, tone = "default" }: NoticeProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View
      accessibilityLiveRegion="polite"
      accessibilityRole={tone === "danger" ? "alert" : undefined}
      style={[styles.base, styles[tone], style]}
      testID={testID}
    >
      <Text tone={titleToneByNotice[tone]} variant="label" weight="bold">
        {title}
      </Text>
      {description ? <Text tone="secondary">{description}</Text> : null}
    </View>
  );
}
