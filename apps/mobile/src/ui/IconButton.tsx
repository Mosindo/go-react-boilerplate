import React from "react";
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { Icon, type IconName } from "./Icon";

type Props = {
  icon: IconName;
  label: string;
  onPress?: () => void;
  size?: number;
  tone?: "plain" | "primary" | "surface" | "scrim";
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function IconButton({ icon, label, onPress, size = 44, tone = "plain", disabled, style, testID }: Props) {
  const { colors } = useTheme();
  const bg = { plain: "transparent", primary: colors.primary, surface: colors.surface, scrim: "rgba(0,0,0,0.35)" }[tone];
  const fg = { plain: colors.text, primary: colors.onPrimary, surface: colors.text, scrim: "#FFFFFF" }[tone];
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      disabled={disabled}
      hitSlop={8}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        { width: size, height: size, borderRadius: size / 2, backgroundColor: bg },
        tone === "surface" ? { borderWidth: StyleSheet.hairlineWidth, borderColor: colors.border, shadowColor: colors.shadow } : null,
        tone === "surface" ? styles.shadow : null,
        pressed ? { transform: [{ scale: 0.94 }] } : null,
        disabled ? { opacity: 0.4 } : null,
        style
      ]}
    >
      <Icon color={fg} name={icon} size={Math.round(size * 0.5)} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { alignItems: "center", justifyContent: "center" },
  shadow: { shadowOpacity: 0.12, shadowRadius: 14, shadowOffset: { width: 0, height: 6 }, elevation: 4 }
});
