import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text, spacing } from "../ui";

export type SectionProps = {
  action?: ReactNode;
  children?: ReactNode;
  style?: StyleProp<ViewStyle>;
  subtitle?: string;
  title: string;
};

export function Section({ action, children, style, subtitle, title }: SectionProps) {
  return (
    <View style={[styles.root, style]}>
      <View style={styles.header}>
        <View style={styles.copy}>
          <Text accessibilityRole="header" variant="heading">
            {title}
          </Text>
          {subtitle ? (
            <Text tone="muted" variant="label">
              {subtitle}
            </Text>
          ) : null}
        </View>
        {action ? <View>{action}</View> : null}
      </View>
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { marginBottom: spacing.xl, gap: spacing.md },
  header: { flexDirection: "row", justifyContent: "space-between", alignItems: "center", gap: spacing.md },
  copy: { flex: 1, gap: spacing.xxs }
});
