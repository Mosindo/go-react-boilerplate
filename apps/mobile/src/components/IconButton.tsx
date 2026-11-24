import React from "react";
import { Pressable, StyleSheet } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { colors, layout } from "../theme/tokens";

type Props = {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  size?: number;
  tone?: "neutral" | "primary" | "plain";
  disabled?: boolean;
  testID?: string;
};

export function IconButton({ icon, label, onPress, size = 44, tone = "neutral", disabled, testID }: Props) {
  const bg = tone === "primary" ? colors.primary : tone === "neutral" ? colors.surfaceRaised : "transparent";
  const fg = tone === "primary" ? colors.onPrimary : colors.text;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      disabled={disabled}
      hitSlop={layout.hitSlop}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        { width: size, height: size, borderRadius: size / 2, backgroundColor: bg },
        tone === "neutral" && styles.bordered,
        pressed && styles.pressed,
        disabled && styles.disabled
      ]}
    >
      <Ionicons name={icon} size={Math.round(size * 0.48)} color={fg} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { alignItems: "center", justifyContent: "center" },
  bordered: { borderWidth: 1, borderColor: colors.border },
  pressed: { transform: [{ scale: 0.94 }], opacity: 0.9 },
  disabled: { opacity: 0.4 }
});
