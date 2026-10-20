import React from "react";
import { Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Badge } from "./Badge";
import { Card } from "./Card";
import { Text } from "./Text";
import { colors, spacing } from "./tokens";

export type NotificationItemProps = {
  body: string;
  createdAtLabel: string;
  isRead: boolean;
  onPress?: () => void;
  style?: StyleProp<ViewStyle>;
  title: string;
  type: string;
};

export function NotificationItem({
  body,
  createdAtLabel,
  isRead,
  onPress,
  style,
  title,
  type
}: NotificationItemProps) {
  return (
    <Pressable disabled={!onPress} onPress={onPress}>
      <Card style={[styles.card, style]} variant={isRead ? "muted" : "accent"}>
        <View style={styles.header}>
          <Text style={styles.title} variant="heading" weight="bold">
            {title}
          </Text>
          <Badge label={isRead ? "Read" : "Unread"} size="sm" variant={isRead ? "muted" : "primary"} />
        </View>
        <Text style={styles.type} tone="secondary" variant="eyebrow" weight="bold">
          {type}
        </Text>
        <Text style={styles.body}>{body}</Text>
        <Text style={styles.meta} tone="muted" variant="caption">
          {createdAtLabel}
        </Text>
      </Card>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    marginBottom: spacing.md
  },
  header: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    gap: spacing.md,
    marginBottom: spacing.sm
  },
  title: {
    flex: 1,
    color: colors.text
  },
  type: {
    marginBottom: spacing.sm
  },
  body: {
    color: colors.text,
    lineHeight: 22
  },
  meta: {
    marginTop: spacing.md
  }
});
