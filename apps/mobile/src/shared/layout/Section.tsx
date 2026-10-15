import React, { type ReactNode } from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Text, spacing } from "../ui";

type SectionProps = {
  title?: string;
  description?: string;
  children: ReactNode;
  style?: StyleProp<ViewStyle>;
};

export function Section({ children, description, style, title }: SectionProps) {
  return (
    <View style={[{ gap: spacing.md }, style]}>
      {title || description ? (
        <View style={{ gap: spacing.xxs }}>
          {title ? (
            <Text accessibilityRole="header" tone="muted" variant="eyebrow" weight="semibold">
              {title}
            </Text>
          ) : null}
          {description ? (
            <Text tone="muted" variant="label">
              {description}
            </Text>
          ) : null}
        </View>
      ) : null}
      {children}
    </View>
  );
}
