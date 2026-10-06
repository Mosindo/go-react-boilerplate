import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Button, Text, spacing, useTheme } from "../ui";

type EmptyViewProps = {
  icon?: keyof typeof Ionicons.glyphMap;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function EmptyView({ icon = "heart-outline", title, message, actionLabel, onAction, style, testID }: EmptyViewProps) {
  const { colors } = useTheme();
  return (
    <View style={[styles.root, style]} testID={testID}>
      <View style={[styles.icon, { backgroundColor: colors.primarySoft }]}>
        <Ionicons color={colors.primary} name={icon} size={30} />
      </View>
      <Text style={styles.center} variant="heading">
        {title}
      </Text>
      {message ? (
        <Text style={styles.center} tone="muted">
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button fullWidth={false} label={actionLabel} onPress={onAction} variant="secondary" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xxl },
  icon: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center" },
  center: { textAlign: "center" }
});
