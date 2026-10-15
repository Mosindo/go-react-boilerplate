import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { errorMessage } from "../../i18n/errors";
import { Button, Text, spacing } from "../ui";

type ErrorViewProps = {
  /** Raw error; its API code is translated to a friendly message. */
  error?: unknown;
  message?: string;
  title?: string;
  onRetry?: () => void;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function ErrorView({ error, message, onRetry, style, testID, title = "Un souci est survenu" }: ErrorViewProps) {
  return (
    <View
      accessibilityRole="alert"
      style={[{ alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl }, style]}
      testID={testID}
    >
      <Text style={{ textAlign: "center" }} variant="heading" weight="bold">
        {title}
      </Text>
      <Text style={{ textAlign: "center", maxWidth: 320 }} tone="muted">
        {message ?? errorMessage(error)}
      </Text>
      {onRetry ? <Button label="Réessayer" onPress={onRetry} size="sm" variant="secondary" /> : null}
    </View>
  );
}
