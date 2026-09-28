import React, { type ReactNode } from "react";
import { ScrollView, StyleSheet, View, type StyleProp, type ViewStyle, type RefreshControlProps } from "react-native";
import type { Edge } from "react-native-safe-area-context";
import { spacing } from "../../theme";
import { SafeAreaLayout } from "./SafeAreaLayout";

export type ScreenContainerProps = {
  children: ReactNode;
  /** Wrap content in a keyboard friendly ScrollView. */
  scroll?: boolean;
  refreshControl?: React.ReactElement<RefreshControlProps>;
  edges?: Edge[];
  /** Screen sits below a native header: skip the top inset to avoid doubling it. */
  underHeader?: boolean;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

const MAX_CONTENT_WIDTH = 560;
const UNDER_HEADER_EDGES: Edge[] = ["bottom", "left", "right"];

export function ScreenContainer({
  children,
  scroll = false,
  refreshControl,
  edges,
  underHeader = false,
  contentStyle,
  testID
}: ScreenContainerProps) {
  return (
    <SafeAreaLayout edges={edges ?? (underHeader ? UNDER_HEADER_EDGES : undefined)}>
      {scroll ? (
        <ScrollView
          contentContainerStyle={[styles.content, contentStyle]}
          keyboardShouldPersistTaps="handled"
          refreshControl={refreshControl}
          testID={testID}
        >
          {children}
        </ScrollView>
      ) : (
        <View style={[styles.content, styles.fill, contentStyle]} testID={testID}>
          {children}
        </View>
      )}
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  content: {
    width: "100%",
    maxWidth: MAX_CONTENT_WIDTH,
    alignSelf: "center",
    padding: spacing.lg,
    gap: spacing.lg
  }
});
