import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { colors, radii, spacing } from "../theme/tokens";
import { AppText } from "./AppText";

export function Section({ title, children, footer }: { title?: string; children: ReactNode; footer?: string }) {
  return (
    <View style={styles.section}>
      {title ? (
        <AppText variant="overline" color={colors.textSubtle} style={styles.title}>
          {title}
        </AppText>
      ) : null}
      <View style={styles.card}>{children}</View>
      {footer ? (
        <AppText variant="caption" color={colors.textSubtle} style={styles.footer}>
          {footer}
        </AppText>
      ) : null}
    </View>
  );
}

type RowProps = {
  label: string;
  value?: string;
  icon?: keyof typeof Ionicons.glyphMap;
  onPress?: () => void;
  destructive?: boolean;
  testID?: string;
};

export function Row({ label, value, icon, onPress, destructive, testID }: RowProps) {
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      disabled={!onPress}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [styles.row, pressed && styles.pressed]}
    >
      {icon ? <Ionicons name={icon} size={20} color={destructive ? colors.danger : colors.textMuted} /> : null}
      <AppText style={styles.rowLabel} color={destructive ? colors.danger : colors.text}>
        {label}
      </AppText>
      {value ? (
        <AppText variant="caption" color={colors.textMuted} numberOfLines={1} style={styles.value}>
          {value}
        </AppText>
      ) : null}
      {onPress ? <Ionicons name="chevron-forward" size={18} color={colors.textSubtle} /> : null}
    </Pressable>
  );
}

type ToggleRowProps = { label: string; description?: string; value: boolean; onChange: (v: boolean) => void; disabled?: boolean; testID?: string };

export function ToggleRow({ label, description, value, onChange, disabled, testID }: ToggleRowProps) {
  return (
    <View style={styles.row}>
      <View style={styles.rowLabel}>
        <AppText>{label}</AppText>
        {description ? (
          <AppText variant="caption" color={colors.textMuted}>
            {description}
          </AppText>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        value={value}
        onValueChange={onChange}
        disabled={disabled}
        testID={testID}
        trackColor={{ true: colors.primary, false: colors.borderStrong }}
        thumbColor={colors.white}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: spacing.xs },
  title: { marginLeft: spacing.xs },
  card: { backgroundColor: colors.surface, borderRadius: radii.md, borderWidth: 1, borderColor: colors.border, overflow: "hidden" },
  footer: { marginHorizontal: spacing.xs },
  row: {
    minHeight: 54,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  rowLabel: { flex: 1, gap: 2 },
  value: { maxWidth: "45%" },
  pressed: { backgroundColor: colors.surfacePressed }
});
