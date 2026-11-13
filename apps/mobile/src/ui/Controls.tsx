import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type RowProps = {
  label: string;
  description?: string;
  icon?: keyof typeof Ionicons.glyphMap;
  onPress?: () => void;
  destructive?: boolean;
  right?: React.ReactNode;
  testID?: string;
};

export function ListRow({ label, description, icon, onPress, destructive, right, testID }: RowProps) {
  const { colors, spacing } = useTheme();
  const body = (
    <View style={[styles.row, { paddingVertical: spacing.sm + 2, gap: spacing.sm }]}>
      {icon ? <Ionicons name={icon} size={22} color={destructive ? colors.danger : colors.textMuted} /> : null}
      <View style={styles.flex}>
        <Text weight="medium" tone={destructive ? "danger" : "default"}>
          {label}
        </Text>
        {description ? (
          <Text variant="caption" tone="muted">
            {description}
          </Text>
        ) : null}
      </View>
      {right ?? (onPress ? <Ionicons name="chevron-forward" size={18} color={colors.textSubtle} /> : null)}
    </View>
  );
  if (!onPress) {
    return body;
  }
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => ({ opacity: pressed ? 0.6 : 1 })}
    >
      {body}
    </Pressable>
  );
}

type SwitchRowProps = Omit<RowProps, "right" | "onPress"> & {
  value: boolean;
  onValueChange: (v: boolean) => void;
  disabled?: boolean;
};

export function SwitchRow({ value, onValueChange, disabled, ...row }: SwitchRowProps) {
  const { colors } = useTheme();
  return (
    <ListRow
      {...row}
      right={
        <Switch
          testID={row.testID}
          accessibilityLabel={row.label}
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

type StepperProps = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  unit?: string;
  onChange: (value: number) => void;
  testID?: string;
};

/** Accessible +/- control, easier to use precisely than a slider. */
export function Stepper({ label, value, min, max, step = 1, unit = "", onChange, testID }: StepperProps) {
  const { colors, radii } = useTheme();
  const button = (icon: "remove" | "add", next: number, disabled: boolean) => (
    <Pressable
      testID={testID ? `${testID}-${icon}` : undefined}
      accessibilityRole="button"
      accessibilityLabel={`${icon === "add" ? "Augmenter" : "Diminuer"} ${label}`}
      disabled={disabled}
      onPress={() => onChange(Math.min(max, Math.max(min, next)))}
      style={[styles.stepBtn, { borderColor: colors.border, borderRadius: radii.pill, opacity: disabled ? 0.4 : 1 }]}
    >
      <Ionicons name={icon} size={20} color={colors.text} />
    </Pressable>
  );
  return (
    <View style={[styles.row, styles.stepper]} accessibilityLabel={`${label} : ${value}${unit}`}>
      <Text weight="medium" style={styles.flex}>
        {label}
      </Text>
      {button("remove", value - step, value <= min)}
      <Text weight="semibold" style={styles.stepValue}>
        {value}
        {unit}
      </Text>
      {button("add", value + step, value >= max)}
    </View>
  );
}

export function Badge({ count }: { count: number }) {
  const { colors } = useTheme();
  if (count <= 0) {
    return null;
  }
  return (
    <View style={[styles.badge, { backgroundColor: colors.primary }]}>
      <Text variant="caption" weight="bold" tone="onPrimary">
        {count > 99 ? "99+" : count}
      </Text>
    </View>
  );
}

export function Divider() {
  const { colors } = useTheme();
  return <View style={{ height: StyleSheet.hairlineWidth, backgroundColor: colors.border }} />;
}

export function ProgressBar({ value }: { value: number }) {
  const { colors } = useTheme();
  return (
    <View
      style={[styles.track, { backgroundColor: colors.surfaceAlt }]}
      accessibilityRole="progressbar"
      accessibilityValue={{ min: 0, max: 100, now: Math.round(value * 100) }}
    >
      <View style={[styles.fill, { width: `${Math.round(value * 100)}%`, backgroundColor: colors.primary }]} />
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  row: { flexDirection: "row", alignItems: "center" },
  stepper: { gap: 12, paddingVertical: 8 },
  stepBtn: { width: 40, height: 40, borderWidth: 1, alignItems: "center", justifyContent: "center" },
  stepValue: { minWidth: 64, textAlign: "center" },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: 11,
    paddingHorizontal: 6,
    alignItems: "center",
    justifyContent: "center"
  },
  track: { height: 4, borderRadius: 2, overflow: "hidden" },
  fill: { height: 4, borderRadius: 2 }
});
