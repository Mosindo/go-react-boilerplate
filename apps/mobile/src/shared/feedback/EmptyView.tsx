import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";
import { Text } from "../ui/Text";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";

type EmptyViewProps = {
  actionLabel?: string;
  message?: string;
  onAction?: () => void;
  style?: StyleProp<ViewStyle>;
  testID?: string;
  title: string;
};

const makeStyles = (t: Theme) => ({
  wrap: { width: "100%" as const },
  card: {
    width: "100%" as const,
    maxWidth: 520,
    alignSelf: "center" as const,
    alignItems: "center" as const,
    gap: t.spacing.sm
  },
  center: { textAlign: "center" as const }
});

export function EmptyView({
  actionLabel,
  message,
  onAction,
  style,
  testID,
  title
}: EmptyViewProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View style={[styles.wrap, style]} testID={testID}>
      <Card padding="lg" style={styles.card} variant="muted">
        <Text style={styles.center} variant="heading" weight="bold">
          {title}
        </Text>
        {message ? (
          <Text style={styles.center} tone="muted">
            {message}
          </Text>
        ) : null}
        {actionLabel && onAction ? (
          <Button label={actionLabel} onPress={onAction} size="sm" variant="secondary" />
        ) : null}
      </Card>
    </View>
  );
}
