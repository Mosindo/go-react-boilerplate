import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import { Button, Text, spacing, type IconName } from "../ui";

type EmptyViewProps = {
  icon?: IconName;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  style?: StyleProp<ViewStyle>;
};

export function EmptyView({ icon = "heart-outline", title, message, actionLabel, onAction, style }: EmptyViewProps) {
  const { colors } = useTheme();
  return (
    <View style={[styles.box, style]}>
      <View style={[styles.iconWrap, { backgroundColor: colors.primarySoft }]}>
        <Ionicons color={colors.primary} name={icon} size={34} />
      </View>
      <Text align="center" variant="heading">
        {title}
      </Text>
      {message ? (
        <Text align="center" tone="muted">
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button onPress={onAction}>{actionLabel}</Button> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  box: { alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl, flex: 1 },
  iconWrap: { width: 76, height: 76, borderRadius: 38, alignItems: "center", justifyContent: "center" }
});
