import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";
import { Text } from "../ui/Text";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";

type ErrorViewProps = {
  actionLabel?: string;
  message: string;
  onAction?: () => void;
  style?: StyleProp<ViewStyle>;
  testID?: string;
  title?: string;
  compact?: boolean;
};

const makeStyles = (t: Theme) => ({
  card: { gap: t.spacing.md },
  cardCompact: {
    flexDirection: "row" as const,
    alignItems: "center" as const,
    justifyContent: "space-between" as const
  },
  copy: { flex: 1, gap: t.spacing.xxs }
});

export function ErrorView({
  actionLabel = "Try again",
  compact = false,
  message,
  onAction,
  style,
  testID,
  title = "Something went wrong"
}: ErrorViewProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <Card
      accessibilityRole="alert"
      padding={compact ? "sm" : "md"}
      style={[styles.card, compact ? styles.cardCompact : null, style]}
      testID={testID}
    >
      <View style={styles.copy}>
        <Text tone="danger" variant={compact ? "label" : "heading"} weight="bold">
          {title}
        </Text>
        <Text tone="muted" variant={compact ? "caption" : "body"}>
          {message}
        </Text>
      </View>
      {onAction ? <Button label={actionLabel} onPress={onAction} size="sm" variant="outline" /> : null}
    </Card>
  );
}
