import React, { type ReactNode } from "react";
import {
  Pressable,
  StyleSheet,
  View,
  type PressableProps,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { Card, type CardVariant } from "./Card";
import { Text } from "./Text";
import { colors, spacing } from "./tokens";

export type ListItemProps = Omit<PressableProps, "children" | "style"> & {
  title: string;
  subtitle?: string;
  leading?: ReactNode;
  trailing?: ReactNode;
  variant?: CardVariant;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function ListItem({
  disabled = false,
  leading,
  onPress,
  style,
  subtitle,
  title,
  trailing,
  variant = "default",
  ...props
}: ListItemProps) {
  const content = (
    <Card padding="md" style={[styles.card, disabled ? styles.disabled : null, style]} variant={variant}>
      {leading ? <View style={styles.leading}>{leading}</View> : null}
      <View style={styles.copy}>
        <Text style={styles.title} variant="label" weight="bold">
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={1} style={styles.subtitle} tone="muted">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {trailing ? <View style={styles.trailing}>{trailing}</View> : null}
    </Card>
  );

  if (!onPress) {
    return content;
  }

  return (
    <Pressable disabled={disabled} onPress={onPress} {...props}>
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md
  },
  disabled: {
    opacity: 0.7
  },
  leading: {
    alignItems: "center",
    justifyContent: "center"
  },
  copy: {
    flex: 1,
    gap: spacing.xs
  },
  title: {
    color: colors.text
  },
  subtitle: {
    color: colors.textMuted
  },
  trailing: {
    marginLeft: spacing.sm
  }
});
