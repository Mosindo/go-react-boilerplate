import React, { type ReactNode } from "react";
import { type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { useTheme } from "../../theme";

type SafeAreaLayoutProps = {
  children: ReactNode;
  edges?: Edge[];
  style?: StyleProp<ViewStyle>;
};

export function SafeAreaLayout({ children, edges, style }: SafeAreaLayoutProps) {
  const palette = useTheme();
  return (
    <SafeAreaView edges={edges} style={[{ flex: 1, backgroundColor: palette.background }, style]}>
      {children}
    </SafeAreaView>
  );
}
