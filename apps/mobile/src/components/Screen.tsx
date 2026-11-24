import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { colors, layout, spacing } from "../theme/tokens";

type Props = {
  children: ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  footer?: ReactNode;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Screen({ children, scroll, edges = ["top"], footer, contentStyle, testID }: Props) {
  const body = scroll ? (
    <ScrollView
      contentContainerStyle={[styles.scrollContent, contentStyle]}
      keyboardShouldPersistTaps="handled"
      showsVerticalScrollIndicator={false}
    >
      <View style={styles.column}>{children}</View>
    </ScrollView>
  ) : (
    <View style={[styles.fill, styles.column, contentStyle]}>{children}</View>
  );
  return (
    <SafeAreaView edges={edges} style={styles.safe} testID={testID}>
      <KeyboardAvoidingView style={styles.fill} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        {body}
        {footer ? <View style={[styles.footer, styles.column]}>{footer}</View> : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: colors.background },
  fill: { flex: 1 },
  column: { width: "100%", maxWidth: layout.maxContentWidth, alignSelf: "center" },
  scrollContent: { paddingHorizontal: spacing.lg, paddingTop: spacing.md, paddingBottom: spacing.xxl, flexGrow: 1 },
  footer: { paddingHorizontal: spacing.lg, paddingTop: spacing.sm, paddingBottom: spacing.md, gap: spacing.sm }
});
