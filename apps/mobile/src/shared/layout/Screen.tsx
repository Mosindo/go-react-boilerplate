import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { type Edge } from "react-native-safe-area-context";
import { colors, spacing } from "../ui/tokens";
import { SafeAreaLayout } from "./SafeAreaLayout";

type ScreenProps = {
  children: ReactNode;
  /** Wrap content in a keyboard-aware ScrollView (forms). */
  scroll?: boolean;
  edges?: Edge[];
  padded?: boolean;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Screen({ children, scroll = false, edges = ["top", "left", "right"], padded = true, contentStyle, testID }: ScreenProps) {
  const padding = padded ? styles.padded : null;
  return (
    <SafeAreaLayout edges={edges} style={styles.root}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        {scroll ? (
          <ScrollView
            contentContainerStyle={[styles.scrollContent, padding, contentStyle]}
            keyboardShouldPersistTaps="handled"
            showsVerticalScrollIndicator={false}
            testID={testID}
          >
            {children}
          </ScrollView>
        ) : (
          <View style={[styles.flex, padding, contentStyle]} testID={testID}>
            {children}
          </View>
        )}
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.background },
  flex: { flex: 1 },
  padded: { paddingHorizontal: spacing.lg, paddingTop: spacing.md },
  scrollContent: { flexGrow: 1, paddingBottom: spacing.xxxl, gap: spacing.lg }
});
