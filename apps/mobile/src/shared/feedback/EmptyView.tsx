import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import { Button, Text, spacing } from "../ui";

type EmptyViewProps = {
  title: string;
  message?: string;
  icon?: keyof typeof Ionicons.glyphMap;
  actionLabel?: string;
  onAction?: () => void;
  children?: ReactNode;
  style?: StyleProp<ViewStyle>;
};

export function EmptyView({ title, message, icon = "heart-outline", actionLabel, onAction, children, style }: EmptyViewProps) {
  const { colors } = useTheme();
  return (
    <View style={[styles.root, style]}>
      <View style={[styles.icon, { backgroundColor: colors.primarySoft }]}>
        <Ionicons color={colors.primary} name={icon} size={30} />
      </View>
      <Text accessibilityRole="header" style={styles.center} variant="heading" weight="bold">
        {title}
      </Text>
      {message ? (
        <Text style={styles.center} tone="muted">
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} /> : null}
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  icon: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center" },
  center: { textAlign: "center" }
});
