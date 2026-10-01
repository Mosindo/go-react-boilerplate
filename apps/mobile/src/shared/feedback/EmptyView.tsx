import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Button, Text, spacing, useTheme } from "../ui";

type EmptyViewProps = {
  icon?: React.ComponentProps<typeof Ionicons>["name"];
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  style?: StyleProp<ViewStyle>;
};

export function EmptyView({ icon = "sparkles-outline", title, message, actionLabel, onAction, style }: EmptyViewProps) {
  const { colors } = useTheme();
  return (
    <View style={[styles.root, style]}>
      <View style={[styles.iconWrap, { backgroundColor: colors.surfaceAccent }]}>
        <Ionicons color={colors.primary} name={icon} size={30} />
      </View>
      <Text align="center" variant="heading">
        {title}
      </Text>
      {message ? (
        <Text align="center" tone="muted">
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button fullWidth={false} label={actionLabel} onPress={onAction} variant="secondary" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  iconWrap: { width: 68, height: 68, borderRadius: 34, alignItems: "center", justifyContent: "center" }
});
