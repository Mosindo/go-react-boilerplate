import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { spacing } from "../ui";
import { SafeAreaLayout } from "./SafeAreaLayout";

type ScreenContainerProps = {
  children: ReactNode;
  centered?: boolean;
  contentMaxWidth?: number;
  contentStyle?: StyleProp<ViewStyle>;
  edges?: ("top" | "right" | "bottom" | "left")[];
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

/** Padded, width-capped page body (tablet / web friendly). */
export function ScreenContainer({
  centered = false,
  children,
  contentMaxWidth = 640,
  contentStyle,
  edges,
  style,
  testID
}: ScreenContainerProps) {
  return (
    <SafeAreaLayout edges={edges} style={style}>
      <View style={[styles.outer, centered && styles.centered]}>
        <View style={[styles.content, centered && styles.contentCentered, { maxWidth: contentMaxWidth }, contentStyle]} testID={testID}>
          {children}
        </View>
      </View>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  outer: { flex: 1, paddingHorizontal: spacing.lg, paddingTop: spacing.md },
  centered: { justifyContent: "center" },
  content: { flex: 1, width: "100%", alignSelf: "center" },
  contentCentered: { flex: 0 }
});
