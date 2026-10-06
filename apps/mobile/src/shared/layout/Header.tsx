import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text, spacing } from "../ui";

type HeaderProps = {
  title: string;
  subtitle?: string;
  action?: ReactNode;
  style?: StyleProp<ViewStyle>;
};

export function Header({ action, subtitle, style, title }: HeaderProps) {
  return (
    <View style={[styles.root, style]}>
      <View style={styles.copy}>
        <Text accessibilityRole="header" variant="title">
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
