import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { Button } from "../ui/Button";
import { Text } from "../ui/Text";

type Props = { message?: string; onRetry?: () => void; title?: string };

export function ErrorView({ message, onRetry, title = "Oups, un souci est survenu" }: Props) {
  return (
    <View style={styles.center} testID="error-view">
      <Text variant="heading" center>
        {title}
      </Text>
      <Text tone="muted" center>
        {message ?? "Veuillez réessayer dans un instant."}
      </Text>
      {onRetry ? <Button label="Réessayer" variant="secondary" onPress={onRetry} /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl }
});
