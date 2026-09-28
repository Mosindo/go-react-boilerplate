import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { Button } from "../ui/Button";
import { Text } from "../ui/Text";

export type ErrorViewProps = { message: string; onRetry?: () => void; title?: string };

export function ErrorView({ message, onRetry, title = "Something went wrong" }: ErrorViewProps) {
  return (
    <View accessibilityRole="alert" style={styles.wrap}>
      <Text center variant="heading">
        {title}
      </Text>
      <Text center tone="muted">
        {message}
      </Text>
      {onRetry ? <Button label="Try again" onPress={onRetry} variant="secondary" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl }
});
