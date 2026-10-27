import React from "react";
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { radii, usePalette } from "../theme";
import { Text } from "./Text";

type Props = {
  label: string;
  onPress: () => void;
  variant?: "primary" | "secondary" | "ghost" | "danger";
  loading?: boolean;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Button({
  label,
  onPress,
  variant = "primary",
  loading,
  disabled,
  style,
  testID
}: Props) {
  const p = usePalette();
  const inactive = disabled || loading;
  const bg =
    variant === "primary"
      ? p.primary
      : variant === "danger"
        ? p.danger
        : variant === "secondary"
          ? p.surfaceAlt
          : "transparent";
  const fg = variant === "primary" ? p.primaryText : variant === "danger" ? "#FFFFFF" : p.text;
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      disabled={inactive}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        { backgroundColor: bg, opacity: inactive ? 0.55 : pressed ? 0.85 : 1 },
        style
      ]}
    >
      {loading ? (
        <ActivityIndicator color={fg} />
      ) : (
        <Text variant="label" style={{ color: fg, fontSize: 16 }}>
          {label}
        </Text>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: 52,
    borderRadius: radii.pill,
    paddingHorizontal: 22,
    alignItems: "center",
    justifyContent: "center"
  }
});
