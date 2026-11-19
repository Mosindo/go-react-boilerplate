import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";
import { spacing } from "../theme/tokens";

type Props = {
  children: React.ReactNode;
  scroll?: boolean;
  padded?: boolean;
  edges?: Edge[];
  footer?: React.ReactNode;
  contentStyle?: StyleProp<ViewStyle>;
};

const MAX_WIDTH = 640;

export function Screen({ children, scroll, padded = true, edges = ["top", "left", "right"], footer, contentStyle }: Props) {
  const { colors } = useTheme();
  const inner = [styles.inner, padded ? styles.padded : null, contentStyle];
  return (
    <SafeAreaView edges={edges} style={[styles.root, { backgroundColor: colors.background }]}>
      <KeyboardAvoidingView style={styles.root} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        {scroll ? (
          <ScrollView contentContainerStyle={[styles.scrollContent, ...inner]} keyboardShouldPersistTaps="handled">
            {children}
          </ScrollView>
        ) : (
          <View style={[styles.root, ...inner]}>{children}</View>
        )}
        {footer ? (
          <View style={[styles.footer, { borderTopColor: colors.border, backgroundColor: colors.background }]}>{footer}</View>
        ) : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  inner: { width: "100%", maxWidth: MAX_WIDTH, alignSelf: "center" },
  padded: { paddingHorizontal: spacing.lg },
  scrollContent: { flexGrow: 1, paddingBottom: spacing.xl },
  footer: {
    width: "100%",
    maxWidth: MAX_WIDTH,
    alignSelf: "center",
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth
  }
});
