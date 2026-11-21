import React from "react";
import { Pressable, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { makeStyles, useTheme, type ThemeColors } from "../theme";
import { Icon, type IconName } from "./Icon";

type IconButtonProps = Omit<PressableProps, "style"> & {
  icon: IconName;
  accessibilityLabel: string;
  size?: number;
  tone?: keyof ThemeColors;
  filled?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function IconButton({ icon, size = 44, tone = "text", filled = false, style, ...props }: IconButtonProps) {
  const styles = useStyles();
  const theme = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      hitSlop={8}
      style={({ pressed }) => [
        styles.base,
        { width: size, height: size, borderRadius: size / 2 },
        filled ? { backgroundColor: theme.colors[tone] } : null,
        pressed ? styles.pressed : null,
        style
      ]}
      {...props}
    >
      <Icon name={icon} size={Math.round(size * 0.48)} color={filled ? "textOnPrimary" : tone} />
    </Pressable>
  );
}

const useStyles = makeStyles(() => ({
  base: { alignItems: "center", justifyContent: "center" },
  pressed: { opacity: 0.7, transform: [{ scale: 0.96 }] }
}));
