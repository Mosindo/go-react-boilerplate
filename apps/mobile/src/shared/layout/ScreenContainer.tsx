import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { type Edge } from "react-native-safe-area-context";
import { useTheme } from "../../theme";
import { spacing } from "../ui";
import { SafeAreaLayout } from "./SafeAreaLayout";

type ScreenContainerProps = {
  children: ReactNode;
  /** Max width keeps the layout readable on tablets and in the browser. */
  contentMaxWidth?: number;
  edges?: Edge[];
  padded?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function ScreenContainer({ children, contentMaxWidth = 560, edges, padded = true, style, testID }: ScreenContainerProps) {
  const { colors } = useTheme();
  return (
    <SafeAreaLayout edges={edges}>
      <View style={[styles.outer, { backgroundColor: colors.background }]}>
        <View style={[styles.content, padded ? styles.padded : null, { maxWidth: contentMaxWidth }, style]} testID={testID}>
          {children}
        </View>
      </View>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  outer: { flex: 1 },
  content: { flex: 1, width: "100%", alignSelf: "center" },
  padded: { paddingHorizontal: spacing.lg, paddingTop: spacing.md }
});
