import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, type StyleProp, type ViewStyle } from "react-native";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";
import { SafeAreaLayout } from "./SafeAreaLayout";

type KeyboardScreenProps = {
  children: ReactNode;
  contentStyle?: StyleProp<ViewStyle>;
  edges?: ("top" | "right" | "bottom" | "left")[];
  maxWidth?: number;
  testID?: string;
};

const makeStyles = (t: Theme) => ({
  flex: { flex: 1 },
  content: {
    flexGrow: 1,
    padding: t.spacing.lg,
    gap: t.spacing.lg,
    width: "100%" as const,
    alignSelf: "center" as const
  }
});

/** Scrollable, keyboard-avoiding screen for forms. */
export function KeyboardScreen({ children, contentStyle, edges, maxWidth = 560, testID }: KeyboardScreenProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <SafeAreaLayout edges={edges}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView
          contentContainerStyle={[styles.content, { maxWidth }, contentStyle]}
          keyboardDismissMode={Platform.OS === "ios" ? "interactive" : "on-drag"}
          keyboardShouldPersistTaps="handled"
          testID={testID}
        >
          {children}
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}
