import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, spacing } from "../theme";
import { Text } from "./Text";

type IconName = React.ComponentProps<typeof Ionicons>["name"];

export function Section({ title, children, footer }: { title?: string; children: ReactNode; footer?: string }) {
  return (
    <View style={styles.section}>
      {title ? (
        <Text variant="overline" tone="subtle" style={styles.sectionTitle}>
          {title}
        </Text>
      ) : null}
      <View style={styles.group}>{children}</View>
      {footer ? (
        <Text variant="caption" tone="subtle" style={styles.footer}>
          {footer}
        </Text>
      ) : null}
    </View>
  );
}

type RowProps = {
  icon?: IconName;
  label: string;
  value?: string;
  onPress?: () => void;
  destructive?: boolean;
  testID?: string;
};

export function Row({ icon, label, value, onPress, destructive, testID }: RowProps) {
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [styles.row, pressed ? styles.pressed : null]}
    >
      {icon ? <Ionicons name={icon} size={20} color={destructive ? colors.danger : colors.textMuted} /> : null}
      <Text style={styles.rowLabel} tone={destructive ? "danger" : "default"}>
        {label}
      </Text>
      {value ? (
        <Text tone="subtle" variant="label" numberOfLines={1} style={styles.value}>
          {value}
        </Text>
      ) : null}
      {onPress ? <Ionicons name="chevron-forward" size={18} color={colors.textSubtle} /> : null}
    </Pressable>
  );
}

type SwitchRowProps = {
  label: string;
  description?: string;
  value: boolean;
  onValueChange: (value: boolean) => void;
  disabled?: boolean;
  testID?: string;
};

export function SwitchRow({ label, description, value, onValueChange, disabled, testID }: SwitchRowProps) {
  return (
    <View style={styles.row}>
      <View style={styles.rowLabel}>
        <Text>{label}</Text>
        {description ? (
          <Text variant="caption" tone="subtle">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        testID={testID}
        accessibilityLabel={label}
        value={value}
        disabled={disabled}
        onValueChange={onValueChange}
        trackColor={{ false: colors.borderStrong, true: colors.accent }}
        thumbColor={colors.white}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  section: {
    gap: spacing.sm,
    marginBottom: spacing.xl
  },
  sectionTitle: {
    marginLeft: spacing.xs
  },
  group: {
    borderRadius: radii.lg,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border,
    overflow: "hidden"
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    minHeight: 54,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  pressed: {
    backgroundColor: colors.surfacePressed
  },
  rowLabel: {
    flex: 1,
    gap: 2
  },
  value: {
    maxWidth: "45%"
  },
  footer: {
    marginHorizontal: spacing.xs
  }
});
