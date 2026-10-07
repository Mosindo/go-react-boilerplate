import React, { type ReactNode } from "react";
import { StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { useTheme } from "../../theme/useTheme";

type SafeAreaLayoutProps = {
  children: ReactNode;
  edges?: Edge[];
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function SafeAreaLayout({
  children,
  edges = ["top", "right", "bottom", "left"],
  style,
  testID
}: SafeAreaLayoutProps) {
  const { colors } = useTheme();
  return (
    <SafeAreaView
      edges={edges}
      testID={testID}
      style={[styles.safeArea, { backgroundColor: colors.background }, style]}
    >
      {children}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1
  }
});
