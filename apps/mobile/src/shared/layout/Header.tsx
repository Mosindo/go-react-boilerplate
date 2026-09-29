import React, { type ReactNode } from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "../ui/Text";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";

type HeaderProps = {
  action?: ReactNode;
  leading?: ReactNode;
  eyebrow?: string;
  subtitle?: string;
  title: string;
  centered?: boolean;
  style?: StyleProp<ViewStyle>;
};

const makeStyles = (t: Theme) => ({
  root: {
    flexDirection: "row" as const,
    alignItems: "flex-start" as const,
    justifyContent: "space-between" as const,
    gap: t.spacing.md
  },
  rootCentered: { justifyContent: "center" as const },
  copy: { flex: 1, gap: t.spacing.xs },
  copyCentered: { alignItems: "center" as const }
});

export function Header({ action, centered = false, eyebrow, leading, subtitle, style, title }: HeaderProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View style={[styles.root, centered ? styles.rootCentered : null, style]}>
      {leading ? <View>{leading}</View> : null}
      <View style={[styles.copy, centered ? styles.copyCentered : null]}>
        {eyebrow ? (
          <Text tone="primary" variant="eyebrow" weight="bold">
            {eyebrow}
          </Text>
        ) : null}
        <Text accessibilityRole="header" variant="title" weight="bold">
          {title}
        </Text>
        {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
      </View>
      {action ? <View>{action}</View> : null}
    </View>
  );
}
