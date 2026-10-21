import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import type { Edge } from "react-native-safe-area-context";
import { spacing } from "../../theme";
import { SafeAreaLayout } from "./SafeAreaLayout";

type Props = {
  children: ReactNode;
  /** Wrap content in a ScrollView (forms, settings). */
  scroll?: boolean;
  edges?: Edge[];
  contentMaxWidth?: number;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

/** Safe-area page with a readable max width, so the app also looks right on tablets and the web. */
export function ScreenContainer({ children, scroll, edges, contentMaxWidth = 640, contentStyle, testID }: Props) {
  const inner = (
    <View style={[styles.content, { maxWidth: contentMaxWidth }, contentStyle]} testID={testID}>
      {children}
    </View>
  );
  return (
    <SafeAreaLayout edges={edges}>
      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        {scroll ? (
          <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled" showsVerticalScrollIndicator={false}>
            {inner}
          </ScrollView>
        ) : (
          <View style={styles.outer}>{inner}</View>
        )}
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  outer: { flex: 1, alignItems: "center" },
  scroll: { flexGrow: 1, alignItems: "center" },
  content: { width: "100%", flex: 1, padding: spacing.lg, gap: spacing.lg }
});
