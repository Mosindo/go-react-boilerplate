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

export function ErrorView({ title = "Un problème est survenu", message, actionLabel = "Réessayer", onAction, style }: ErrorViewProps) {
  const { colors } = useTheme();
  return (
    <View accessibilityRole="alert" style={[styles.box, style]}>
      <Ionicons color={colors.danger} name="cloud-offline-outline" size={40} />
      <Text align="center" variant="heading">
        {title}
      </Text>
      <Text align="center" tone="muted">
        {message}
      </Text>
      {onAction ? (
        <Button onPress={onAction} variant="outline">
          {actionLabel}
        </Button>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  box: { alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl, flex: 1 }
});
