import React, { type ComponentProps } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: Variant;
  size?: "md" | "lg";
  loading?: boolean;
  icon?: ComponentProps<typeof Ionicons>["name"];
  fullWidth?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function Button({ label, variant = "primary", size = "lg", loading, icon, fullWidth, disabled, style, ...props }: ButtonProps) {
  const theme = useTheme();
  const c = theme.colors;
  const isDisabled = disabled || loading;
  const palette: Record<Variant, { bg: string; pressed: string; fg: string; border: string }> = {
    primary: { bg: c.primary, pressed: c.primaryPressed, fg: c.onPrimary, border: c.primary },
    secondary: { bg: c.surface, pressed: c.surfaceAlt, fg: c.text, border: c.borderStrong },
    ghost: { bg: "transparent", pressed: c.surfaceAlt, fg: c.text, border: "transparent" },
    danger: { bg: c.dangerSoft, pressed: c.dangerSoft, fg: c.danger, border: c.dangerSoft }
  };
  const p = palette[variant];

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!isDisabled, busy: !!loading }}
      disabled={isDisabled}
      style={({ pressed }) => [
        styles.base,
        {
          minHeight: size === "lg" ? 54 : 44,
          paddingHorizontal: size === "lg" ? theme.spacing.xl : theme.spacing.md,
          backgroundColor: pressed ? p.pressed : p.bg,
          borderColor: p.border,
          borderRadius: theme.radii.pill,
          opacity: isDisabled ? 0.5 : 1
        },
        fullWidth ? styles.full : null,
        style
      ]}
      {...props}
    >
      <View style={styles.row}>
        {loading ? <ActivityIndicator color={p.fg} size="small" /> : icon ? <Ionicons color={p.fg} name={icon} size={18} /> : null}
        <Text color={p.fg} variant={size === "lg" ? "heading" : "label"} style={size === "lg" ? styles.lgLabel : null}>
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { alignItems: "center", justifyContent: "center", borderWidth: 1 },
  full: { alignSelf: "stretch" },
  row: { flexDirection: "row", alignItems: "center", gap: 8 },
  lgLabel: { fontSize: 16 }
});
