import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text, spacing } from "../ui";

type HeaderProps = {
  action?: ReactNode;
  leading?: ReactNode;
  eyebrow?: string;
  subtitle?: string;
  title: string;
  centered?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function Header({ action, centered = false, eyebrow, leading, subtitle, style, title }: HeaderProps) {
  return (
    <View style={[styles.root, centered && styles.rootCentered, style]}>
      {leading ? <View>{leading}</View> : null}
      <View style={[styles.copy, centered && styles.copyCentered]}>
        {eyebrow ? (
          <Text tone="primary" variant="eyebrow" weight="bold">
            {eyebrow}
          </Text>
        ) : null}
        <Text accessibilityRole="header" align={centered ? "center" : undefined} variant="title">
          {title}
        </Text>
        {subtitle ? (
          <Text align={centered ? "center" : undefined} tone="muted">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {action ? <View>{action}</View> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.md, paddingBottom: spacing.md },
  rootCentered: { justifyContent: "center" },
  copy: { flex: 1, gap: spacing.xxs },
  copyCentered: { alignItems: "center" }
});
