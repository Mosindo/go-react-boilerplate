import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { layout, spacing, useTheme } from "../theme";

type ScreenProps = {
  children: React.ReactNode;
  scroll?: boolean;
  padded?: boolean;
  edges?: Edge[];
  contentStyle?: StyleProp<ViewStyle>;
  footer?: React.ReactNode;
  testID?: string;
};

/** Safe-area aware page with a centred, width-capped column (comfortable on tablets and web). */
export function Screen({ children, scroll, padded = true, edges = ["top"], contentStyle, footer, testID }: ScreenProps) {
  const { colors } = useTheme();
  const inner = [styles.column, padded ? styles.padded : null, contentStyle];
  return (
    <SafeAreaView edges={edges} style={[styles.root, { backgroundColor: colors.background }]} testID={testID}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.root}>
        {scroll ? (
          <ScrollView contentContainerStyle={[styles.scrollContent]} keyboardShouldPersistTaps="handled">
            <View style={inner}>{children}</View>
          </ScrollView>
        ) : (
          <View style={[styles.root, ...inner]}>{children}</View>
        )}
        {footer ? <View style={[styles.column, styles.footer]}>{footer}</View> : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  scrollContent: { flexGrow: 1, paddingBottom: spacing.xxl },
  column: { width: "100%", maxWidth: layout.maxContentWidth, alignSelf: "center" },
  padded: { paddingHorizontal: spacing.lg },
  footer: { paddingHorizontal: spacing.lg, paddingTop: spacing.sm, paddingBottom: spacing.md, gap: spacing.sm }
});
