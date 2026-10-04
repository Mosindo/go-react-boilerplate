import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { MIN_TOUCH, radii, spacing, useColors } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export function SectionTitle({ children }: { children: string }) {
  return (
    <Text accessibilityRole="header" tone="muted" variant="small" weight="semibold" style={styles.sectionTitle}>
      {children.toUpperCase()}
    </Text>
  );
}

export function Card({ children }: { children: ReactNode }) {
  const colors = useColors();
  return <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>{children}</View>;
}

type RowProps = {
  label: string;
  icon?: IconName;
  detail?: string;
  destructive?: boolean;
  onPress: () => void;
};

export function NavRow({ label, icon, detail, destructive, onPress }: RowProps) {
  const colors = useColors();
  return (
    <Pressable
      accessibilityLabel={detail ? `${label}, ${detail}` : label}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [styles.row, pressed && { backgroundColor: colors.surfaceMuted }]}
    >
      {icon ? <Icon color={destructive ? colors.danger : colors.primary} name={icon} size={22} /> : null}
      <Text style={styles.grow} tone={destructive ? "danger" : "default"} weight="medium">
        {label}
      </Text>
      {detail ? (
        <Text tone="muted" variant="small">
          {detail}
        </Text>
      ) : null}
      {!destructive ? <Icon color={colors.textMuted} name="chevron-forward" size={18} /> : null}
    </Pressable>
  );
}

type SwitchRowProps = {
  label: string;
  description?: string;
  value: boolean;
  onValueChange: (value: boolean) => void;
  disabled?: boolean;
};

export function SwitchRow({ label, description, value, onValueChange, disabled }: SwitchRowProps) {
  const colors = useColors();
  return (
    <View style={styles.row}>
      <View style={styles.grow}>
        <Text weight="medium">{label}</Text>
        {description ? (
          <Text tone="muted" variant="small">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        disabled={disabled}
        onValueChange={onValueChange}
        thumbColor="#FFFFFF"
        trackColor={{ true: colors.primary, false: colors.border }}
        value={value}
      />
    </View>
  );
}

type StepperProps = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  unit?: string;
  onChange: (value: number) => void;
};

export function Stepper({ label, value, min, max, step = 1, unit, onChange }: StepperProps) {
  const colors = useColors();
  const button = (name: "remove" | "add", delta: number, disabled: boolean, accessibilityLabel: string) => (
    <Pressable
      accessibilityLabel={accessibilityLabel}
      accessibilityRole="button"
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={() => onChange(Math.min(max, Math.max(min, value + delta)))}
      style={[styles.stepButton, { backgroundColor: colors.primarySoft }, disabled && { opacity: 0.4 }]}
    >
      <Icon color={colors.primary} name={name} size={20} />
    </Pressable>
  );
  return (
    <View style={styles.stepper}>
      <Text style={styles.grow} weight="medium">
        {label}
      </Text>
      {button("remove", -step, value <= min, `Decrease ${label}`)}
      <Text
        accessibilityLabel={`${label}: ${value}${unit ? ` ${unit}` : ""}`}
        center
        style={styles.stepValue}
        weight="semibold"
      >
        {value}
        {unit ? ` ${unit}` : ""}
      </Text>
      {button("add", step, value >= max, `Increase ${label}`)}
    </View>
  );
}

const styles = StyleSheet.create({
  sectionTitle: { marginTop: spacing.sm, letterSpacing: 0.6 },
  card: { borderRadius: radii.lg, borderWidth: 1, overflow: "hidden" },
  row: {
    minHeight: MIN_TOUCH + 12,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  grow: { flex: 1 },
  stepper: { flexDirection: "row", alignItems: "center", gap: spacing.xs, minHeight: MIN_TOUCH + 8 },
  stepButton: {
    width: MIN_TOUCH,
    height: MIN_TOUCH,
    borderRadius: MIN_TOUCH / 2,
    alignItems: "center",
    justifyContent: "center"
  },
  stepValue: { minWidth: 64 }
});
