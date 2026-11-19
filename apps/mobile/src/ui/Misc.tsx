import React from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { radii, spacing } from "../theme/tokens";
import { Icon, type IconName } from "./Icon";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

export function Card({ children, style }: { children: React.ReactNode; style?: object }) {
  const { colors } = useTheme();
  return <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }, style]}>{children}</View>;
}

export function SectionTitle({ children }: { children: string }) {
  return (
    <Text variant="caption" tone="subtle" style={styles.sectionTitle} accessibilityRole="header">
      {children.toUpperCase()}
    </Text>
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

export function ListRow({ icon, label, value, onPress, destructive, testID }: RowProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      disabled={!onPress}
      testID={testID}
      style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
    >
      {icon ? <Icon name={icon} size={20} color={destructive ? colors.danger : colors.textMuted} /> : null}
      <Text style={styles.rowLabel} tone={destructive ? "danger" : "default"}>
        {label}
      </Text>
      {value ? (
        <Text tone="muted" numberOfLines={1} style={styles.rowValue}>
          {value}
        </Text>
      ) : null}
      {onPress ? <Icon name="chevron-forward" size={18} color={colors.textSubtle} /> : null}
    </Pressable>
  );
}

export function ToggleRow({
  label,
  description,
  value,
  onChange,
  testID
}: {
  label: string;
  description?: string;
  value: boolean;
  onChange: (v: boolean) => void;
  testID?: string;
}) {
  const { colors } = useTheme();
  return (
    <View style={styles.row}>
      <View style={styles.rowLabel}>
        <Text>{label}</Text>
        {description ? (
          <Text variant="caption" tone="muted">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        value={value}
        onValueChange={onChange}
        trackColor={{ true: colors.primary, false: colors.borderStrong }}
        thumbColor="#FFFFFF"
        testID={testID}
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
  suffix?: string;
  onChange: (v: number) => void;
};

export function Stepper({ label, value, min, max, step = 1, suffix = "", onChange }: StepperProps) {
  return (
    <View style={styles.row}>
      <Text style={styles.rowLabel}>{label}</Text>
      <IconButton
        icon="remove"
        label={`Diminuer ${label}`}
        tone="default"
        size={36}
        disabled={value <= min}
        onPress={() => onChange(Math.max(min, value - step))}
      />
      <Text variant="label" style={styles.stepValue} accessibilityLiveRegion="polite">
        {value}
        {suffix}
      </Text>
      <IconButton
        icon="add"
        label={`Augmenter ${label}`}
        tone="default"
        size={36}
        disabled={value >= max}
        onPress={() => onChange(Math.min(max, value + step))}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  card: { borderRadius: radii.lg, borderWidth: 1, paddingHorizontal: spacing.md, paddingVertical: spacing.xs },
  sectionTitle: { letterSpacing: 1.2, marginTop: spacing.lg, marginBottom: spacing.xs, marginLeft: spacing.xs },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, minHeight: 52, paddingVertical: spacing.xs, borderRadius: radii.sm },
  rowLabel: { flex: 1 },
  rowValue: { maxWidth: "45%" },
  stepValue: { minWidth: 64, textAlign: "center" }
});
