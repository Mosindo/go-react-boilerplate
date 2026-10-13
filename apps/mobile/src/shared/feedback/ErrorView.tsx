import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import { Button, Text, spacing } from "../ui";

type ErrorViewProps = {
  title?: string;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
  style?: StyleProp<ViewStyle>;
};

export function ErrorView({ title = "Oups, un souci", message, actionLabel = "Réessayer", onAction, style }: ErrorViewProps) {
  const { colors } = useTheme();
  return (
    <View accessibilityRole="alert" style={[styles.root, style]}>
      <Ionicons color={colors.danger} name="cloud-offline-outline" size={36} />
      <Text style={styles.center} variant="heading" weight="bold">
        {title}
      </Text>
      <Text style={styles.center} tone="muted">
        {message}
      </Text>
      {onAction ? <Button label={actionLabel} onPress={onAction} variant="outline" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  center: { textAlign: "center" }
});
