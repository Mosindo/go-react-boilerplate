import React, { type ReactNode } from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Button, Text, spacing } from "../ui";

type EmptyViewProps = {
  actionLabel?: string;
  icon?: ReactNode;
  message?: string;
  onAction?: () => void;
  style?: StyleProp<ViewStyle>;
  testID?: string;
  title: string;
};

export function EmptyView({ actionLabel, icon, message, onAction, style, testID, title }: EmptyViewProps) {
  return (
    <View
      accessibilityRole="summary"
      style={[{ alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl }, style]}
      testID={testID}
    >
      {icon}
      <Text style={{ textAlign: "center" }} variant="heading" weight="bold">
        {title}
      </Text>
      {message ? (
        <Text style={{ textAlign: "center", maxWidth: 320 }} tone="muted">
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} size="sm" variant="secondary" /> : null}
    </View>
  );
}
