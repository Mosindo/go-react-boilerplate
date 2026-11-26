import React from "react";
import { Platform, Pressable, StyleSheet, Switch, View } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export function Card({ children, testID }: { children: React.ReactNode; testID?: string }) {
  const { colors } = useTheme();
  return (
    <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]} testID={testID}>
      {children}
    </View>
  );
}

type ListRowProps = {
  icon: IconName;
  label: string;
  detail?: string;
  onPress?: () => void;
  destructive?: boolean;
  testID?: string;
};

export function ListRow({ icon, label, detail, onPress, destructive, testID }: ListRowProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
      testID={testID}
    >
      <Icon color={destructive ? colors.danger : colors.textMuted} name={icon} />
      <View style={styles.rowText}>
        <Text tone={destructive ? "danger" : "default"} variant="bodyStrong">
          {label}
        </Text>
        {detail ? (
          <Text tone="muted" variant="caption">
            {detail}
          </Text>
        ) : null}
      </View>
      {onPress && !destructive ? <Icon color={colors.textSubtle} name="chevron-forward" size={18} /> : null}
    </Pressable>
  );
}

type SwitchRowProps = {
  label: string;
  detail?: string;
  value: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
  testID?: string;
};

export function SwitchRow({ label, detail, value, onChange, disabled, testID }: SwitchRowProps) {
  const { colors } = useTheme();
  // react-native-web ignores thumbColor for the "on" state and uses activeThumbColor instead.
  const webThumb: object = Platform.OS === "web" ? { activeThumbColor: colors.surface } : {};
  return (
    <View style={styles.row}>
      <View style={styles.rowText}>
        <Text variant="bodyStrong">{label}</Text>
        {detail ? (
          <Text tone="muted" variant="caption">
            {detail}
          </Text>
        ) : null}
      </View>
      <Switch
        {...webThumb}
        accessibilityLabel={label}
        disabled={disabled}
        onValueChange={onChange}
        testID={testID}
        thumbColor={colors.surface}
        trackColor={{ false: colors.border, true: colors.primary }}
        value={value}
      />
    </View>
  );
}

export function SectionTitle({ children }: { children: string }) {
  return (
    <Text style={styles.section} tone="subtle" variant="overline">
      {children}
    </Text>
  );
}

const styles = StyleSheet.create({
  card: { borderRadius: radii.lg, borderWidth: 1, padding: spacing.xs, overflow: "hidden" },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.sm,
    borderRadius: radii.md,
    minHeight: 52
  },
  rowText: { flex: 1, gap: 2 },
  section: { marginTop: spacing.lg, marginBottom: spacing.xs, marginLeft: spacing.xxs }
});
