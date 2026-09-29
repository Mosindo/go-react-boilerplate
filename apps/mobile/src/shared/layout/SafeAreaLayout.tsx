import React, { type ReactNode } from "react";
import { type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";

type SafeAreaLayoutProps = {
  children: ReactNode;
  edges?: Edge[];
  style?: StyleProp<ViewStyle>;
};

const makeStyles = (t: Theme) => ({
  safeArea: { flex: 1, backgroundColor: t.colors.background }
});

export function SafeAreaLayout({
  children,
  edges = ["top", "right", "bottom", "left"],
  style
}: SafeAreaLayoutProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <SafeAreaView edges={edges} style={[styles.safeArea, style]}>
      {children}
    </SafeAreaView>
  );
}
