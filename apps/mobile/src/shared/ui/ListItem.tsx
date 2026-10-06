import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { spacing } from "./tokens";

export type ListItemProps = {
  title: string;
  subtitle?: string;
  leading?: ReactNode;
  trailing?: ReactNode;
  /** Shows a chevron and makes the row pressable. */
  onPress?: () => void;
  destructive?: boolean;
  testID?: string;
};

export function ListItem({ title, subtitle, leading, trailing, onPress, destructive = false, testID }: ListItemProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={({ pressed }) => [styles.root, { borderBottomColor: colors.border, opacity: pressed ? 0.7 : 1 }]}
      testID={testID}
    >
      {leading}
      <View style={styles.copy}>
        <Text tone={destructive ? "danger" : "default"} weight="medium">
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={1} tone="muted" variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {trailing}
      {onPress && !trailing ? <Ionicons color={colors.textSubtle} name="chevron-forward" size={18} /> : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  copy: { flex: 1, gap: 2 }
});
