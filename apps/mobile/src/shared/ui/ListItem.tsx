import React, { type ReactNode } from "react";
import { Pressable, View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { controls, spacing } from "./tokens";

export type ListItemProps = {
  title: string;
  subtitle?: string;
  leading?: ReactNode;
  trailing?: ReactNode;
  onPress?: () => void;
  destructive?: boolean;
  bordered?: boolean;
  accessibilityLabel?: string;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function ListItem({
  accessibilityLabel,
  bordered = false,
  destructive = false,
  leading,
  onPress,
  style,
  subtitle,
  testID,
  title,
  trailing
}: ListItemProps) {
  const palette = useTheme();
  const content = (
    <>
      {leading}
      <View style={{ flex: 1, gap: 2 }}>
        <Text numberOfLines={1} tone={destructive ? "danger" : "default"} weight="semibold">
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={2} tone="muted" variant="label">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {trailing}
    </>
  );
  const rowStyle: ViewStyle = {
    minHeight: controls.minTouch + 12,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: bordered ? 1 : 0,
    borderBottomColor: palette.border
  };

  if (!onPress) {
    return (
      <View style={[rowStyle, style]} testID={testID}>
        {content}
      </View>
    );
  }
  return (
    <Pressable
      accessibilityLabel={accessibilityLabel ?? title}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [rowStyle, pressed ? { opacity: 0.7 } : null, style]}
      testID={testID}
    >
      {content}
    </Pressable>
  );
}
