import React from "react";
import { Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { shadow } from "../theme/tokens";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type Props = {
  icon: IconName;
  label: string;
  onPress: () => void;
  size?: number;
  tone?: "default" | "primary" | "plain";
  badge?: number;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function IconButton({ icon, label, onPress, size = 44, tone = "plain", badge, disabled, style, testID }: Props) {
  const { colors } = useTheme();
  const bg = tone === "primary" ? colors.primary : tone === "default" ? colors.surface : "transparent";
  const fg = tone === "primary" ? colors.onPrimary : colors.text;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      disabled={disabled}
      hitSlop={8}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        { width: size, height: size, borderRadius: size / 2, backgroundColor: bg, opacity: disabled ? 0.4 : pressed ? 0.75 : 1 },
        tone !== "plain" ? shadow : null,
        tone === "default" ? { borderWidth: 1, borderColor: colors.border } : null,
        style
      ]}
    >
      <Icon name={icon} size={Math.round(size * 0.48)} color={fg} />
      {badge ? (
        <View style={[styles.badge, { backgroundColor: colors.accent, borderColor: colors.background }]}>
          <Text variant="caption" style={{ color: "#1A0F15", fontSize: 11 }} weight="700">
            {badge > 9 ? "9+" : badge}
          </Text>
        </View>
      ) : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { alignItems: "center", justifyContent: "center" },
  badge: {
    position: "absolute",
    top: -2,
    right: -2,
    minWidth: 20,
    height: 20,
    borderRadius: 10,
    borderWidth: 2,
    paddingHorizontal: 4,
    alignItems: "center",
    justifyContent: "center"
  }
});
