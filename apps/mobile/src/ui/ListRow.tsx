import React from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { spacing, useTheme } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export function ListRow({
  icon,
  title,
  subtitle,
  onPress,
  destructive,
  right,
  last
}: {
  icon?: IconName;
  title: string;
  subtitle?: string;
  onPress?: () => void;
  destructive?: boolean;
  right?: React.ReactNode;
  last?: boolean;
}) {
  const { colors } = useTheme();
  const tint = destructive ? colors.danger : colors.text;
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      accessibilityLabel={title}
      onPress={onPress}
      disabled={!onPress}
      style={({ pressed }) => [
        styles.row,
        {
          backgroundColor: pressed ? colors.surfaceMuted : "transparent",
          borderBottomColor: colors.border,
          borderBottomWidth: last ? 0 : StyleSheet.hairlineWidth
        }
      ]}
    >
      {icon ? <Icon name={icon} size={20} color={destructive ? colors.danger : colors.textMuted} /> : null}
      <View style={styles.text}>
        <Text variant="bodyStrong" style={{ color: tint }}>
          {title}
        </Text>
        {subtitle ? (
          <Text variant="caption" tone="muted">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right ?? (onPress ? <Icon name="chevron-forward" size={18} color={colors.textSubtle} /> : null)}
    </Pressable>
  );
}

export function SwitchRow({
  title,
  subtitle,
  value,
  onValueChange,
  last,
  disabled
}: {
  title: string;
  subtitle?: string;
  value: boolean;
  onValueChange: (v: boolean) => void;
  last?: boolean;
  disabled?: boolean;
}) {
  const { colors } = useTheme();
  return (
    <ListRow
      title={title}
      subtitle={subtitle}
      last={last}
      right={
        <Switch
          accessibilityLabel={title}
          value={value}
          disabled={disabled}
          onValueChange={onValueChange}
          trackColor={{ true: colors.primary, false: colors.borderStrong }}
          thumbColor="#FFFFFF"
        />
      }
    />
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingHorizontal: spacing.md, paddingVertical: spacing.sm + 2, minHeight: 56 },
  text: { flex: 1, gap: 2 }
});
