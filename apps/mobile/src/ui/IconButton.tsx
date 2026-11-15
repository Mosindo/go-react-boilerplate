import React from "react";
import { Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme, type Palette } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type Props = {
  icon: IconName;
  label: string;
  onPress: () => void;
  size?: number;
  tone?: keyof Palette;
  background?: keyof Palette;
  badge?: number;
  disabled?: boolean;
  testID?: string;
  style?: StyleProp<ViewStyle>;
};

export function IconButton({
  icon,
  label,
  onPress,
  size = 44,
  tone = "text",
  background = "surfaceAlt",
  badge,
  disabled,
  testID,
  style
}: Props) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      hitSlop={8}
      disabled={disabled}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        { width: size, height: size, borderRadius: size / 2, backgroundColor: colors[background], opacity: disabled ? 0.5 : 1 },
        pressed && { transform: [{ scale: 0.92 }] },
        style
      ]}
    >
      <Icon name={icon} size={Math.round(size * 0.48)} color={tone} />
      {badge ? (
        <View style={[styles.badge, { backgroundColor: colors.accent, borderColor: colors.background }]}>
          <Text variant="micro" style={{ color: "#fff" }}>
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
    paddingHorizontal: 5,
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 2
  }
});
