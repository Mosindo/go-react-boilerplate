import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { colors, layout, spacing } from "../theme/tokens";

export type ScreenProps = {
  children: React.ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  padded?: boolean;
  footer?: React.ReactNode;
  contentStyle?: StyleProp<ViewStyle>;
  keyboard?: boolean;
  testID?: string;
};

export function Screen({ children, scroll, edges = ["top"], padded = true, footer, contentStyle, keyboard, testID }: ScreenProps) {
  const inner = scroll ? (
    <ScrollView
      contentContainerStyle={[styles.scrollContent, padded && styles.padded, contentStyle]}
      keyboardShouldPersistTaps="handled"
      showsVerticalScrollIndicator={false}
    >
      <View style={styles.column}>{children}</View>
    </ScrollView>
  ) : (
    <View style={[styles.fill, padded && styles.padded, contentStyle]}>
      <View style={[styles.column, styles.fill]}>{children}</View>
    </View>
  );

  const body = (
    <>
      {inner}
      {footer ? (
        <View style={styles.footer}>
          <View style={styles.column}>{footer}</View>
        </View>
      ) : null}
    </>
  );

  return (
    <SafeAreaView style={styles.root} edges={edges} testID={testID}>
      {keyboard ? (
        <KeyboardAvoidingView style={styles.fill} behavior={Platform.OS === "ios" ? "padding" : undefined}>
          {body}
        </KeyboardAvoidingView>
      ) : (
        body
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.background },
  fill: { flex: 1 },
  padded: { paddingHorizontal: spacing.xl, paddingTop: spacing.lg },
  scrollContent: { flexGrow: 1, paddingBottom: spacing.huge },
  column: { width: "100%", maxWidth: layout.maxContentWidth, alignSelf: "center" },
  footer: {
    paddingHorizontal: spacing.xl,
    paddingTop: spacing.md,
    paddingBottom: spacing.lg,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: colors.border,
    backgroundColor: colors.background
  }
});
