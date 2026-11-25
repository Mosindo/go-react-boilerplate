import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { colors, layout, spacing } from "../theme";

type ScreenProps = {
  children: React.ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  padded?: boolean;
  footer?: React.ReactNode;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

/** Root layout of every screen: safe area, keyboard handling, centred column. */
export function Screen({ children, scroll, edges = ["top", "bottom"], padded = true, footer, contentStyle, testID }: ScreenProps) {
  const content = [styles.column, padded && styles.padded, contentStyle];
  return (
    <SafeAreaView edges={edges} style={styles.root} testID={testID}>
      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        {scroll ? (
          <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled" showsVerticalScrollIndicator={false}>
            <View style={content}>{children}</View>
          </ScrollView>
        ) : (
          <View style={[styles.flex, content]}>{children}</View>
        )}
        {footer ? <View style={[styles.column, styles.footer]}>{footer}</View> : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: colors.background
  },
  flex: {
    flex: 1
  },
  scroll: {
    flexGrow: 1
  },
  column: {
    width: "100%",
    maxWidth: layout.maxContentWidth,
    alignSelf: "center"
  },
  padded: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  },
  footer: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.sm,
    paddingBottom: spacing.md,
    gap: spacing.sm
  }
});
