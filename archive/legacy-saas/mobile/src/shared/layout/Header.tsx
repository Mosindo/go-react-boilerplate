import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text, colors, spacing } from "../ui";

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
    <View style={[styles.root, centered ? styles.rootCentered : null, style]}>
      {leading ? <View style={styles.leading}>{leading}</View> : null}
      <View style={[styles.copy, centered ? styles.copyCentered : null]}>
        {eyebrow ? (
          <Text tone="secondary" variant="eyebrow" weight="bold">
            {eyebrow}
          </Text>
        ) : null}
        <Text style={styles.title} variant="title" weight="bold">
          {title}
        </Text>
        {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
      </View>
      {action ? <View style={styles.action}>{action}</View> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flexDirection: "row",
    alignItems: "flex-start",
    justifyContent: "space-between",
    gap: spacing.lg
  },
  rootCentered: {
    justifyContent: "center"
  },
  leading: {
    paddingTop: spacing.md
  },
  copy: {
    flex: 1,
    gap: spacing.sm
  },
  copyCentered: {
    alignItems: "center"
  },
  action: {
    paddingTop: spacing.md
  },
  title: {
    color: colors.text
  }
});
