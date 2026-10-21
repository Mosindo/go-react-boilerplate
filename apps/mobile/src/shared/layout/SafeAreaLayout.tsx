import React, { type ReactNode } from "react";
import { type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { useTheme } from "../../theme";

type Props = { children: ReactNode; edges?: Edge[]; style?: StyleProp<ViewStyle> };

export function SafeAreaLayout({ children, edges = ["top", "right", "bottom", "left"], style }: Props) {
  const theme = useTheme();
  return (
    <SafeAreaView edges={edges} style={[{ flex: 1, backgroundColor: theme.background }, style]}>
      {children}
    </SafeAreaView>
  );
}
