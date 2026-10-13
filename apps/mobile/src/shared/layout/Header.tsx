import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text, spacing } from "../ui";

type HeaderProps = {
  action?: ReactNode;
  leading?: ReactNode;
  eyebrow?: string;
  subtitle?: string;
  title: string;
  style?: StyleProp<ViewStyle>;
};

export function Header({ action, eyebrow, leading, subtitle, style, title }: HeaderProps) {
  return (
    <View style={[styles.root, style]}>
      {leading}
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
      {action}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.md, paddingBottom: spacing.md },
  copy: { flex: 1, gap: spacing.xxs }
});
