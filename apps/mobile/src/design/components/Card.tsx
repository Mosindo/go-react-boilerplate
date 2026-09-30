import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../ThemeProvider";

export function Card({ children, style }: { children: React.ReactNode; style?: StyleProp<ViewStyle> }) {
  const { colors, radii } = useTheme();
  return (
    <View
      style={[
        { backgroundColor: colors.surface, borderRadius: radii.lg, borderWidth: 1, borderColor: colors.border, overflow: "hidden" },
        style
      ]}
    >
      {children}
    </View>
  );
}
