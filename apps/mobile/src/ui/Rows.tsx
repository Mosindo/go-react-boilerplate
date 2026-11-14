import React from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { radii, spacing, type Palette } from "../theme/palette";
import { useStyles, useTheme } from "../theme/ThemeProvider";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export function Section({ title, children, footer }: { title?: string; children: React.ReactNode; footer?: string }) {
  const styles = useStyles(makeStyles);
  return (
    <View style={styles.section}>
      {title ? (
        <Text style={styles.sectionTitle} tone="subtle" variant="overline">
          {title}
        </Text>
      ) : null}
      <View style={styles.card}>{children}</View>
      {footer ? (
        <Text style={styles.footer} tone="subtle" variant="caption">
          {footer}
        </Text>
      ) : null}
    </View>
  );
}

type RowProps = {
  icon?: IconName;
  title: string;
  subtitle?: string;
  value?: string;
  onPress?: () => void;
  destructive?: boolean;
  testID?: string;
};

export function ListRow({ icon, title, subtitle, value, onPress, destructive, testID }: RowProps) {
  const styles = useStyles(makeStyles);
  const { colors } = useTheme();
  const tint = destructive ? colors.danger : colors.text;
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={({ pressed }) => [styles.row, pressed ? { backgroundColor: colors.surfaceMuted } : null]}
      testID={testID}
    >
      {icon ? <Icon color={destructive ? colors.danger : colors.textMuted} name={icon} size={20} /> : null}
      <View style={styles.rowText}>
        <Text style={{ color: tint }} variant="body">
          {title}
        </Text>
        {subtitle ? (
          <Text tone="muted" variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {value ? (
        <Text numberOfLines={1} style={styles.value} tone="muted" variant="caption">
          {value}
        </Text>
      ) : null}
      {onPress && !destructive ? <Icon color={colors.textSubtle} name="chevron-forward" size={18} /> : null}
    </Pressable>
  );
}

type SwitchRowProps = {
  title: string;
  subtitle?: string;
  value: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
};

export function SwitchRow({ title, subtitle, value, onChange, disabled }: SwitchRowProps) {
  const styles = useStyles(makeStyles);
  const { colors } = useTheme();
  return (
    <View style={styles.row}>
      <View style={styles.rowText}>
        <Text>{title}</Text>
        {subtitle ? (
          <Text tone="muted" variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      <Switch
        // react-native-web colours the active thumb separately.
        {...({ activeThumbColor: colors.surface } as object)}
        accessibilityLabel={title}
        disabled={disabled}
        onValueChange={onChange}
        thumbColor={colors.surface}
        trackColor={{ false: colors.borderStrong, true: colors.primary }}
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

/** Accessible numeric stepper (no slider dependency). */
export function Stepper({ label, value, min, max, step = 1, unit, onChange }: StepperProps) {
  const styles = useStyles(makeStyles);
  const { colors } = useTheme();
  const clamp = (v: number) => Math.min(max, Math.max(min, v));
  return (
    <View
      accessibilityActions={[{ name: "increment" }, { name: "decrement" }]}
      accessibilityLabel={label}
      accessibilityRole="adjustable"
      accessibilityValue={{ min, max, now: value, text: `${value}${unit ? ` ${unit}` : ""}` }}
      onAccessibilityAction={(e) => onChange(clamp(value + (e.nativeEvent.actionName === "increment" ? step : -step)))}
      style={styles.row}
    >
      <Text style={styles.rowText}>{label}</Text>
      <Pressable
        accessibilityLabel={`Diminuer ${label}`}
        disabled={value <= min}
        hitSlop={6}
        onPress={() => onChange(clamp(value - step))}
        style={[styles.stepBtn, { opacity: value <= min ? 0.35 : 1 }]}
      >
        <Icon color={colors.text} name="remove" size={18} />
      </Pressable>
      <Text style={styles.stepValue} weight="700">
        {value}
        {unit ? ` ${unit}` : ""}
      </Text>
      <Pressable
        accessibilityLabel={`Augmenter ${label}`}
        disabled={value >= max}
        hitSlop={6}
        onPress={() => onChange(clamp(value + step))}
        style={[styles.stepBtn, { opacity: value >= max ? 0.35 : 1 }]}
      >
        <Icon color={colors.text} name="add" size={18} />
      </Pressable>
    </View>
  );
}

export function Divider() {
  const { colors } = useTheme();
  return <View style={{ height: StyleSheet.hairlineWidth, backgroundColor: colors.border, marginLeft: spacing.md }} />;
}

const makeStyles = (c: Palette) =>
  StyleSheet.create({
    section: { gap: spacing.xs, marginBottom: spacing.lg },
    sectionTitle: { marginLeft: spacing.xs },
    card: { backgroundColor: c.surface, borderRadius: radii.lg, borderWidth: StyleSheet.hairlineWidth, borderColor: c.border, overflow: "hidden" },
    footer: { marginHorizontal: spacing.xs },
    row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, minHeight: 54, paddingHorizontal: spacing.md, paddingVertical: spacing.xs },
    rowText: { flex: 1, gap: 2 },
    value: { maxWidth: 160 },
    stepBtn: {
      width: 36,
      height: 36,
      borderRadius: 18,
      borderWidth: 1,
      borderColor: c.border,
      alignItems: "center",
      justifyContent: "center"
    },
    stepValue: { minWidth: 64, textAlign: "center" }
  });
