import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text, spacing } from "../ui";

type SectionProps = {
  title?: string;
  children: ReactNode;
  style?: StyleProp<ViewStyle>;
};

export function Section({ title, children, style }: SectionProps) {
  return (
    <View style={[styles.root, style]}>
      {title ? (
        <Text accessibilityRole="header" tone="secondary" variant="eyebrow" weight="bold">
          {title}
        </Text>
      ) : null}
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.sm, marginBottom: spacing.xl }
});
