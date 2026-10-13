import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text, spacing } from "../ui";

type SectionProps = {
  title?: string;
  description?: string;
  children: ReactNode;
  style?: StyleProp<ViewStyle>;
};

export function Section({ title, description, children, style }: SectionProps) {
  return (
    <View style={[styles.root, style]}>
      {title ? (
        <View style={styles.heading}>
          <Text accessibilityRole="header" variant="label" weight="bold" tone="muted" style={styles.title}>
            {title}
          </Text>
          {description ? (
            <Text tone="subtle" variant="caption">
              {description}
            </Text>
          ) : null}
        </View>
      ) : null}
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.sm },
  heading: { gap: 2 },
  title: { textTransform: "uppercase", letterSpacing: 1 }
});
