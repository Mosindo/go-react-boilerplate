import React, { type ReactNode } from "react";
import { type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { useTheme } from "../ui";

type SafeAreaLayoutProps = {
  children: ReactNode;
  edges?: Edge[];
  style?: StyleProp<ViewStyle>;
};

export function SafeAreaLayout({ children, edges = ["top", "right", "bottom", "left"], style }: SafeAreaLayoutProps) {
  const { colors } = useTheme();
  return (
    <SafeAreaView edges={edges} style={[{ flex: 1, backgroundColor: colors.background }, style]}>
      {children}
    </SafeAreaView>
  );
}
