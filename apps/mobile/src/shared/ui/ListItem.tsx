import React, { type ReactNode } from "react";
import { Pressable, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { Card, type CardVariant } from "./Card";
import { Text } from "./Text";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type ListItemProps = Omit<PressableProps, "children" | "style"> & {
  title: string;
  subtitle?: string;
  leading?: ReactNode;
  trailing?: ReactNode;
  variant?: CardVariant;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
};

const makeStyles = (t: Theme) => ({
  card: {
    flexDirection: "row" as const,
    alignItems: "center" as const,
    gap: t.spacing.md,
    minHeight: 44
  },
  disabled: { opacity: 0.6 },
  leading: { alignItems: "center" as const, justifyContent: "center" as const },
  copy: { flex: 1, gap: t.spacing.xxs },
  trailing: { marginLeft: t.spacing.sm }
});

export function ListItem({
  accessibilityLabel,
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
  const styles = useThemedStyles(makeStyles);
  const content = (
    <Card
      padding="md"
      style={[styles.card, disabled ? styles.disabled : null, style]}
      variant={variant}
    >
      {leading ? <View style={styles.leading}>{leading}</View> : null}
      <View style={styles.copy}>
        <Text variant="label" weight="bold">
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={2} tone="muted">
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
    <Pressable
      accessibilityLabel={accessibilityLabel ?? (subtitle ? `${title}, ${subtitle}` : title)}
      accessibilityRole="button"
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      {...props}
    >
      {content}
    </Pressable>
  );
}
