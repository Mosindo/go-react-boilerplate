import React from "react";
import { Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Card } from "./Card";
import { Text } from "./Text";
import { colors, radii, spacing } from "./tokens";

export type NotificationItemProps = {
  body: string;
  createdAtLabel: string;
  isRead: boolean;
  onPress?: () => void;
  style?: StyleProp<ViewStyle>;
  title: string;
};

export function NotificationItem({ body, createdAtLabel, isRead, onPress, style, title }: NotificationItemProps) {
  return (
    <Pressable
      accessibilityLabel={`${isRead ? "" : "Non lue. "}${title}. ${body}`}
      accessibilityRole="button"
      disabled={!onPress}
      onPress={onPress}
    >
      <Card padding="md" style={[styles.card, style]} variant={isRead ? "muted" : "accent"}>
        <View style={styles.row}>
          {isRead ? <View style={styles.dotSpacer} /> : <View style={styles.dot} />}
          <View style={styles.copy}>
            <Text variant="label" weight="bold">
              {title}
            </Text>
            <Text tone="secondary">{body}</Text>
            <Text tone="muted" variant="caption">
              {createdAtLabel}
            </Text>
          </View>
        </View>
      </Card>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    marginBottom: spacing.sm
  },
  row: {
    flexDirection: "row",
    gap: spacing.md,
    alignItems: "flex-start"
  },
  dot: {
    width: 10,
    height: 10,
    borderRadius: radii.pill,
    backgroundColor: colors.primary,
    marginTop: 5
  },
  dotSpacer: {
    width: 10
  },
  copy: {
    flex: 1,
    gap: spacing.xxs
  }
});
