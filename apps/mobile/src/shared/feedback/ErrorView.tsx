import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Button, Text, spacing, useTheme } from "../ui";

type ErrorViewProps = {
  title?: string;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
  style?: StyleProp<ViewStyle>;
};

/** Full-area error with a retry action. */
export function ErrorView({ title = "Un problème est survenu", message, actionLabel = "Réessayer", onAction, style }: ErrorViewProps) {
  const { colors } = useTheme();
  return (
    <View accessibilityRole="alert" style={[styles.root, style]}>
      <Ionicons color={colors.danger} name="cloud-offline-outline" size={36} />
      <Text align="center" variant="heading">
        {title}
      </Text>
      <Text align="center" tone="muted">
        {message}
      </Text>
      {onAction ? <Button fullWidth={false} label={actionLabel} onPress={onAction} variant="outline" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({ root: { alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl } });
