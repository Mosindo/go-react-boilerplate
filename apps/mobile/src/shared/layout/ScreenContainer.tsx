import React, { type ReactNode } from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { spacing } from "../ui";
import { SafeAreaLayout } from "./SafeAreaLayout";

type ScreenContainerProps = {
  children: ReactNode;
  centered?: boolean;
  /** Which safe-area edges to pad; tab/stack headers already cover the top. */
  edges?: ("top" | "right" | "bottom" | "left")[];
  padded?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function ScreenContainer({
  centered = false,
  children,
  edges = ["left", "right"],
  padded = true,
  style,
  testID
}: ScreenContainerProps) {
  return (
    <SafeAreaLayout edges={edges}>
      <View
        style={[
          { flex: 1, width: "100%", maxWidth: 640, alignSelf: "center" },
          padded ? { paddingHorizontal: spacing.lg, paddingTop: spacing.md } : null,
          centered ? { justifyContent: "center" } : null,
          style
        ]}
        testID={testID}
      >
        {children}
      </View>
    </SafeAreaLayout>
  );
}
