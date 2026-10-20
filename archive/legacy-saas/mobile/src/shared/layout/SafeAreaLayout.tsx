import React, { type ReactNode } from "react";
import { StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { colors } from "../ui";

type SafeAreaLayoutProps = {
  children: ReactNode;
  edges?: Edge[];
  style?: StyleProp<ViewStyle>;
};

export function SafeAreaLayout({
  children,
  edges = ["top", "right", "bottom", "left"],
  style
}: SafeAreaLayoutProps) {
  return (
    <SafeAreaView edges={edges} style={[styles.safeArea, style]}>
      {children}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background
  }
});
