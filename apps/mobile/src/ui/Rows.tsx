import React, { type ComponentProps, type ReactNode } from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type IconName = ComponentProps<typeof Ionicons>["name"];

export function Section({ title, children, footer }: { title?: string; children: ReactNode; footer?: string }) {
  const theme = useTheme();
  return (
    <View style={styles.section}>
      {title ? (
        <Text style={styles.sectionTitle} tone="muted" variant="label">
          {title.toUpperCase()}
        </Text>
      ) : null}
      <View style={[styles.card, { backgroundColor: theme.colors.surface, borderColor: theme.colors.border, borderRadius: theme.radii.lg }]}>{children}</View>
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
  label: string;
  value?: string;
  onPress?: () => void;
  tone?: "default" | "danger";
  last?: boolean;
  testID?: string;
};

export function Row({ icon, label, value, onPress, tone = "default", last, testID }: RowProps) {
  const theme = useTheme();
  const color = tone === "danger" ? theme.colors.danger : theme.colors.text;
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={({ pressed }) => [
        styles.row,
        !last && { borderBottomWidth: StyleSheet.hairlineWidth, borderColor: theme.colors.border },
        pressed && { opacity: 0.6 }
      ]}
      testID={testID}
    >
      {icon ? <Ionicons color={color} name={icon} size={20} /> : null}
      <Text color={color} style={styles.rowLabel}>
        {label}
      </Text>
      {value ? (
        <Text numberOfLines={1} style={styles.rowValue} tone="muted" variant="small">
          {value}
        </Text>
      ) : null}
      {onPress ? <Ionicons color={theme.colors.textSubtle} name="chevron-forward" size={18} /> : null}
    </Pressable>
  );
}

type ToggleRowProps = {
  label: string;
  description?: string;
  value: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
  last?: boolean;
};

export function ToggleRow({ label, description, value, onChange, disabled, last }: ToggleRowProps) {
  const theme = useTheme();
  return (
    <View style={[styles.row, !last && { borderBottomWidth: StyleSheet.hairlineWidth, borderColor: theme.colors.border }]}>
      <View style={styles.rowLabel}>
        <Text>{label}</Text>
        {description ? (
          <Text tone="muted" variant="caption">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        disabled={disabled}
        onValueChange={onChange}
        thumbColor={theme.colors.surface}
        trackColor={{ false: theme.colors.borderStrong, true: theme.colors.primary }}
        value={value}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: 8, marginBottom: 24 },
  sectionTitle: { paddingHorizontal: 4 },
  card: { borderWidth: 1, overflow: "hidden" },
  footer: { paddingHorizontal: 4 },
  row: { flexDirection: "row", alignItems: "center", gap: 12, paddingHorizontal: 16, paddingVertical: 14, minHeight: 52 },
  rowLabel: { flex: 1, gap: 2 },
  rowValue: { maxWidth: "45%" }
});
