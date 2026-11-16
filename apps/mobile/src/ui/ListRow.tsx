import React from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, spacing } from "../theme/tokens";
import { Text } from "./Text";

export type ListRowProps = {
  icon?: keyof typeof Ionicons.glyphMap;
  title: string;
  subtitle?: string;
  onPress?: () => void;
  tone?: "default" | "danger";
  value?: boolean;
  onValueChange?: (v: boolean) => void;
  testID?: string;
};

export function ListRow({ icon, title, subtitle, onPress, tone = "default", value, onValueChange, testID }: ListRowProps) {
  const color = tone === "danger" ? colors.danger : colors.text;
  const isSwitch = typeof value === "boolean" && onValueChange;
  return (
    <Pressable
      accessibilityRole={isSwitch ? "switch" : "button"}
      accessibilityState={isSwitch ? { checked: value } : undefined}
      accessibilityLabel={title}
      onPress={isSwitch ? () => onValueChange(!value) : onPress}
      testID={testID}
      style={({ pressed }) => [styles.row, pressed && { backgroundColor: colors.surfaceRaised }]}
    >
      {icon ? (
        <View style={styles.icon}>
          <Ionicons name={icon} size={20} color={tone === "danger" ? colors.danger : colors.accent} />
        </View>
      ) : null}
      <View style={styles.texts}>
        <Text variant="bodyStrong" style={{ color }}>
          {title}
        </Text>
        {subtitle ? (
          <Text variant="caption" tone="muted">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {isSwitch ? (
        <Switch value={value} onValueChange={onValueChange} trackColor={{ true: colors.accent, false: colors.border }} thumbColor={colors.text} />
      ) : onPress ? (
        <Ionicons name="chevron-forward" size={18} color={colors.textSubtle} />
      ) : null}
    </Pressable>
  );
}

export function ListGroup({ children }: { children: React.ReactNode }) {
  return <View style={styles.group}>{children}</View>;
}

const styles = StyleSheet.create({
  group: { backgroundColor: colors.surface, borderRadius: radii.lg, borderWidth: 1, borderColor: colors.border, overflow: "hidden" },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.md, minHeight: 60 },
  icon: { width: 36, height: 36, borderRadius: radii.md, backgroundColor: colors.surfaceRaised, alignItems: "center", justifyContent: "center" },
  texts: { flex: 1, gap: 2 }
});
