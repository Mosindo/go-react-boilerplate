import React from "react";
import { Pressable, StyleSheet, View, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

export function Section({ title, children, style }: { title?: string; children: React.ReactNode; style?: ViewStyle }) {
  const { colors, radii } = useTheme();
  return (
    <View style={{ gap: 8 }}>
      {title ? (
        <Text variant="overline" tone="subtle" style={{ paddingHorizontal: 4 }}>
          {title}
        </Text>
      ) : null}
      <View
        style={[
          {
            backgroundColor: colors.surface,
            borderRadius: radii.lg,
            borderWidth: 1,
            borderColor: colors.border,
            overflow: "hidden"
          },
          style
        ]}
      >
        {children}
      </View>
    </View>
  );
}

type RowProps = {
  icon?: keyof typeof Ionicons.glyphMap;
  label: string;
  value?: string;
  onPress?: () => void;
  tone?: "default" | "danger";
  right?: React.ReactNode;
  testID?: string;
  last?: boolean;
};

export function Row({ icon, label, value, onPress, tone = "default", right, testID, last }: RowProps) {
  const { colors } = useTheme();
  const color = tone === "danger" ? colors.danger : colors.text;
  return (
    <Pressable
      testID={testID}
      disabled={!onPress}
      onPress={onPress}
      accessibilityRole={onPress ? "button" : undefined}
      style={({ pressed }) => [
        styles.row,
        { borderBottomColor: colors.border, borderBottomWidth: last ? 0 : StyleSheet.hairlineWidth },
        pressed ? { backgroundColor: colors.surfaceSunken } : null
      ]}
    >
      {icon ? <Ionicons name={icon} size={20} color={tone === "danger" ? colors.danger : colors.textMuted} /> : null}
      <Text style={{ flex: 1, color }}>{label}</Text>
      {value ? <Text tone="muted">{value}</Text> : null}
      {right}
      {onPress && !right ? <Ionicons name="chevron-forward" size={18} color={colors.textSubtle} /> : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: 12, paddingHorizontal: 16, minHeight: 54 }
});
