import React, { type ReactNode } from "react";
import { Pressable, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import { Text, controls, spacing } from "../ui";

type HeaderProps = {
  title: string;
  subtitle?: string;
  onBack?: () => void;
  right?: ReactNode;
  style?: StyleProp<ViewStyle>;
};

export function Header({ onBack, right, style, subtitle, title }: HeaderProps) {
  const palette = useTheme();
  return (
    <View style={[{ minHeight: 56, flexDirection: "row", alignItems: "center", gap: spacing.md, paddingVertical: spacing.sm }, style]}>
      {onBack ? (
        <Pressable
          accessibilityLabel="Retour"
          accessibilityRole="button"
          hitSlop={8}
          onPress={onBack}
          style={{
            width: controls.minTouch,
            height: controls.minTouch,
            alignItems: "center",
            justifyContent: "center",
            marginLeft: -spacing.sm
          }}
        >
          <Ionicons color={palette.text} name="chevron-back" size={26} />
        </Pressable>
      ) : null}
      <View style={{ flex: 1 }}>
        <Text accessibilityRole="header" numberOfLines={1} variant="heading" weight="bold">
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={1} tone="muted" variant="label">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right}
    </View>
  );
}
