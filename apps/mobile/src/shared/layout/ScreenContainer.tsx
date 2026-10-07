import React, { type ReactNode } from "react";
import { useTheme } from "../../theme/useTheme";
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

export function ScreenContainer({
  centered = false,
  children,
  contentMaxWidth = 960,
  contentStyle,
  edges,
  style,
  testID
}: ScreenContainerProps) {
  const { colors } = useTheme();
  return (
    <SafeAreaLayout edges={edges} style={style}>
      <View style={[styles.outer, { backgroundColor: colors.background }, centered ? styles.centered : null]}>
        <View
          style={[
            styles.content,
            centered ? styles.contentCentered : null,
            { maxWidth: contentMaxWidth },
            contentStyle
          ]}
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
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.lg,
    paddingBottom: spacing.lg
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
