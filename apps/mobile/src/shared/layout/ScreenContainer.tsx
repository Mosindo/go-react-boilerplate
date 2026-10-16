import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { colors, spacing } from "../ui";
import { SafeAreaLayout } from "./SafeAreaLayout";

// The floating tab bar is 68px tall and sits 14px above the bottom edge.
const TAB_BAR_CLEARANCE = 68 + spacing.md + spacing.lg;

type ScreenContainerProps = {
  children: ReactNode;
  centered?: boolean;
  contentMaxWidth?: number;
  contentStyle?: StyleProp<ViewStyle>;
  edges?: ("top" | "right" | "bottom" | "left")[];
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function ScreenContainer({
  centered = false,
  children,
  contentMaxWidth = 960,
  contentStyle,
  edges,
  style,
  testID
}: ScreenContainerProps) {
  return (
    <SafeAreaLayout edges={edges} style={style}>
      <View style={[styles.outer, centered ? styles.centered : null]}>
        <View
          style={[styles.content, centered ? styles.contentCentered : null, { maxWidth: contentMaxWidth }, contentStyle]}
          testID={testID}
        >
          {children}
        </View>
      </View>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  outer: {
    flex: 1,
    backgroundColor: colors.background,
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.lg,
    paddingBottom: TAB_BAR_CLEARANCE
  },
  centered: {
    justifyContent: "center"
  },
  content: {
    flex: 1,
    width: "100%",
    alignSelf: "center"
  },
  contentCentered: {
    flex: 0
  }
});
