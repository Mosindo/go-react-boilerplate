import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text, spacing } from "../ui";

export type SectionProps = {
  action?: ReactNode;
  children?: ReactNode;
  eyebrow?: string;
  style?: StyleProp<ViewStyle>;
  subtitle?: string;
  title: string;
};

export function Section({ action, children, eyebrow, style, subtitle, title }: SectionProps) {
  return (
    <View style={[styles.root, style]}>
      <View style={styles.header}>
        <View style={styles.copy}>
          {eyebrow ? (
            <Text style={styles.eyebrow} tone="secondary" variant="eyebrow" weight="bold">
              {eyebrow}
            </Text>
          ) : null}
          <Text style={styles.title} variant="heading" weight="bold">
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

const styles = StyleSheet.create({
  root: {
    marginBottom: spacing.xxl
  },
  header: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "flex-start",
    gap: spacing.md,
    marginBottom: spacing.md
  },
  copy: {
    flex: 1,
    gap: spacing.xs
  },
  eyebrow: {
    marginBottom: spacing.xs
  },
  title: {
    marginBottom: spacing.xs
  }
});
