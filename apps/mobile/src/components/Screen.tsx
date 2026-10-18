import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaLayout } from "../shared/layout";
import { colors, spacing } from "../shared/ui";

type ScreenProps = {
  children: React.ReactNode;
  /** Wrap in a ScrollView (forms, settings). Lists and the deck manage their own scrolling. */
  scroll?: boolean;
  /** Keeps a footer (primary action) pinned below the scrollable content. */
  footer?: React.ReactNode;
  header?: React.ReactNode;
  edges?: ("top" | "right" | "bottom" | "left")[];
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Screen({ children, scroll = false, footer, header, edges, contentStyle, testID }: ScreenProps) {
  return (
    <SafeAreaLayout edges={edges}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        {header}
        {scroll ? (
          <ScrollView contentContainerStyle={[styles.content, contentStyle]} keyboardShouldPersistTaps="handled" style={styles.flex} testID={testID}>
            <View style={styles.inner}>{children}</View>
          </ScrollView>
        ) : (
          <View style={[styles.content, styles.flex, contentStyle]} testID={testID}>
            <View style={[styles.inner, styles.flex]}>{children}</View>
          </View>
        )}
        {footer ? <View style={styles.footer}>{footer}</View> : null}
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { padding: spacing.lg, gap: spacing.lg, flexGrow: 1 },
  inner: { width: "100%", maxWidth: 560, alignSelf: "center", gap: spacing.lg },
  footer: {
    padding: spacing.lg,
    borderTopWidth: 1,
    borderTopColor: colors.border,
    backgroundColor: colors.background,
    alignItems: "center"
  }
});
