import React from "react";
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { useTheme, type Palette } from "../theme";
import { Icon, type IconName } from "./Icon";

type Props = {
  icon: IconName;
  label: string;
  onPress: () => void;
  size?: number;
  tone?: "plain" | "surface" | "accent" | "muted";
  color?: keyof Palette;
  disabled?: boolean;
  testID?: string;
  style?: StyleProp<ViewStyle>;
};

export function IconButton({ icon, label, onPress, size = 44, tone = "plain", color, disabled, testID, style }: Props) {
  const { colors } = useTheme();
  const bg = { plain: "transparent", surface: colors.surface, accent: colors.accent, muted: colors.surfaceAlt }[tone];
  const fg: keyof Palette = color ?? (tone === "accent" ? "textOnAccent" : "text");
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      disabled={disabled}
      hitSlop={8}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        { width: size, height: size, borderRadius: size / 2, backgroundColor: bg, opacity: disabled ? 0.4 : pressed ? 0.7 : 1 },
        tone === "surface" ? { borderWidth: 1, borderColor: colors.border } : null,
        style
      ]}
      testID={testID}
    >
      <Icon color={fg} name={icon} size={Math.round(size * 0.48)} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { alignItems: "center", justifyContent: "center" }
});
