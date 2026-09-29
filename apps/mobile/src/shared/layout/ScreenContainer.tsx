import React, { type ReactNode } from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";
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

const makeStyles = (t: Theme) => ({
  outer: {
    flex: 1,
    backgroundColor: t.colors.background,
    paddingHorizontal: t.spacing.lg,
    paddingTop: t.spacing.lg,
    paddingBottom: t.spacing.lg
  },
  centered: { justifyContent: "center" as const },
  content: { flex: 1, width: "100%" as const, alignSelf: "center" as const },
  contentCentered: { flex: 0 }
});

export function ScreenContainer({
  centered = false,
  children,
  contentMaxWidth = 960,
  contentStyle,
  edges,
  style,
  testID
}: ScreenContainerProps) {
  const styles = useThemedStyles(makeStyles);
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
