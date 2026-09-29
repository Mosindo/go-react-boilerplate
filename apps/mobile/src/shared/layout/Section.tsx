import React, { type ReactNode } from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "../ui/Text";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";

export type SectionProps = {
  action?: ReactNode;
  children?: ReactNode;
  eyebrow?: string;
  style?: StyleProp<ViewStyle>;
  subtitle?: string;
  title: string;
};

const makeStyles = (t: Theme) => ({
  root: { marginBottom: t.spacing.xl },
  header: {
    flexDirection: "row" as const,
    justifyContent: "space-between" as const,
    alignItems: "flex-start" as const,
    gap: t.spacing.md,
    marginBottom: t.spacing.md
  },
  copy: { flex: 1, gap: t.spacing.xxs }
});

export function Section({ action, children, eyebrow, style, subtitle, title }: SectionProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View style={[styles.root, style]}>
      <View style={styles.header}>
        <View style={styles.copy}>
          {eyebrow ? (
            <Text tone="primary" variant="eyebrow" weight="bold">
              {eyebrow}
            </Text>
          ) : null}
          <Text accessibilityRole="header" variant="heading" weight="bold">
            {title}
          </Text>
          {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
        </View>
        {action ? <View>{action}</View> : null}
      </View>
      {children}
    </View>
  );
}
