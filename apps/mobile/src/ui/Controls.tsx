import React from "react";
import { Modal, Platform, Pressable, ScrollView, StyleSheet, Switch, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";
import { radii, spacing } from "../theme/palette";
import { IconButton } from "./Button";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type ListRowProps = {
  icon?: IconName;
  title: string;
  subtitle?: string;
  onPress?: () => void;
  destructive?: boolean;
  right?: React.ReactNode;
  testID?: string;
};

export function ListRow({ icon, title, subtitle, onPress, destructive, right, testID }: ListRowProps) {
  const { colors } = useTheme();
  const tint = destructive ? colors.danger : colors.text;
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      accessibilityLabel={title}
      onPress={onPress}
      disabled={!onPress}
      testID={testID}
      style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
    >
      {icon ? (
        <View style={[styles.rowIcon, { backgroundColor: destructive ? colors.dangerSoft : colors.surfaceMuted }]}>
          <Icon name={icon} size={18} color={tint} />
        </View>
      ) : null}
      <View style={styles.flex}>
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

type ToggleRowProps = {
  title: string;
  subtitle?: string;
  value: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
  testID?: string;
};

export function ToggleRow({ title, subtitle, value, onChange, disabled, testID }: ToggleRowProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.row}>
      <View style={styles.flex}>
        <Text variant="bodyStrong">{title}</Text>
        {subtitle ? (
          <Text variant="caption" tone="muted">
            {subtitle}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={title}
        value={value}
        onValueChange={onChange}
        disabled={disabled}
        testID={testID}
        trackColor={{ true: colors.primary, false: colors.borderStrong }}
        thumbColor="#FFFFFF"
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
  onChange: (value: number) => void;
};

export function Stepper({ label, value, min, max, step = 1, suffix, onChange }: StepperProps) {
  const { colors } = useTheme();
  return (
    <View style={[styles.stepper, { borderColor: colors.border, backgroundColor: colors.surface }]}>
      <Text variant="label" tone="muted" style={styles.flex}>
        {label}
      </Text>
      <IconButton
        icon="remove"
        accessibilityLabel={`Diminuer ${label}`}
        variant="surface"
        size={36}
        disabled={value <= min}
        onPress={() => onChange(Math.max(min, value - step))}
      />
      <Text variant="bodyStrong" style={styles.stepValue} accessibilityLabel={`${label} : ${value}${suffix ?? ""}`}>
        {value}
        {suffix ?? ""}
      </Text>
      <IconButton
        icon="add"
        accessibilityLabel={`Augmenter ${label}`}
        variant="surface"
        size={36}
        disabled={value >= max}
        onPress={() => onChange(Math.min(max, value + step))}
      />
    </View>
  );
}

type SheetProps = {
  visible: boolean;
  title: string;
  onClose: () => void;
  children: React.ReactNode;
};

/** Bottom sheet built on Modal (no extra dependency). */
export function Sheet({ visible, title, onClose, children }: SheetProps) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <Modal visible={visible} transparent animationType={Platform.OS === "web" ? "none" : "slide"} onRequestClose={onClose} statusBarTranslucent>
      <Pressable style={[StyleSheet.absoluteFill, { backgroundColor: colors.overlay }]} onPress={onClose} accessibilityLabel="Fermer" />
      <View style={[styles.sheet, { backgroundColor: colors.background, paddingBottom: insets.bottom + spacing.md }]}>
        <View style={[styles.grabber, { backgroundColor: colors.borderStrong }]} />
        <View style={styles.sheetHeader}>
          <Text variant="heading" style={styles.flex} accessibilityRole="header">
            {title}
          </Text>
          <IconButton icon="close" accessibilityLabel="Fermer" onPress={onClose} />
        </View>
        <ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={styles.sheetContent}>
          {children}
        </ScrollView>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingVertical: spacing.sm, paddingHorizontal: spacing.xs, borderRadius: radii.md },
  rowIcon: { width: 36, height: 36, borderRadius: 18, alignItems: "center", justifyContent: "center" },
  stepper: { flexDirection: "row", alignItems: "center", gap: spacing.sm, borderWidth: 1, borderRadius: radii.md, padding: spacing.xs, paddingLeft: spacing.md },
  stepValue: { minWidth: 56, textAlign: "center" },
  sheet: { position: "absolute", left: 0, right: 0, bottom: 0, maxHeight: "88%", borderTopLeftRadius: radii.xl, borderTopRightRadius: radii.xl, paddingHorizontal: spacing.lg },
  grabber: { alignSelf: "center", width: 40, height: 5, borderRadius: 3, marginTop: spacing.xs },
  sheetHeader: { flexDirection: "row", alignItems: "center", paddingVertical: spacing.xs },
  sheetContent: { gap: spacing.sm, paddingBottom: spacing.md }
});
