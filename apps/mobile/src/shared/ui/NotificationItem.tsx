import React from "react";
import { Pressable, View, type StyleProp, type ViewStyle } from "react-native";
import { Badge } from "./Badge";
import { Card } from "./Card";
import { Text } from "./Text";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type NotificationItemProps = {
  body: string;
  createdAtLabel: string;
  isRead: boolean;
  onPress?: () => void;
  style?: StyleProp<ViewStyle>;
  title: string;
  type: string;
};

const makeStyles = (t: Theme) => ({
  card: { marginBottom: t.spacing.md, gap: t.spacing.xs },
  header: {
    flexDirection: "row" as const,
    justifyContent: "space-between" as const,
    alignItems: "center" as const,
    gap: t.spacing.md
  },
  title: { flex: 1 }
});

export function NotificationItem({ body, createdAtLabel, isRead, onPress, style, title, type }: NotificationItemProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <Pressable accessibilityRole={onPress ? "button" : undefined} disabled={!onPress} onPress={onPress}>
      <Card style={[styles.card, style]} variant={isRead ? "muted" : "accent"}>
        <View style={styles.header}>
          <Text style={styles.title} variant="label" weight="bold">
            {title}
          </Text>
          <Badge label={isRead ? "Read" : "New"} size="sm" variant={isRead ? "muted" : "primary"} />
        </View>
        <Text tone="secondary" variant="eyebrow" weight="bold">
          {type}
        </Text>
        <Text>{body}</Text>
        <Text tone="muted" variant="caption">
          {createdAtLabel}
        </Text>
      </Card>
    </Pressable>
  );
}
