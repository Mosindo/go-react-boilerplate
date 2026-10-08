import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../../theme";
import { spacing } from "../ui";

type ScreenContainerProps = {
  children: ReactNode;
  scroll?: boolean;
  padded?: boolean;
  contentStyle?: StyleProp<ViewStyle>;
  /** Pinned below the content (primary action of a form). */
  footer?: ReactNode;
};

/** Page wrapper: themed background, optional scrolling, keyboard avoidance and a pinned footer. */
export function ScreenContainer({ children, scroll = true, padded = true, contentStyle, footer }: ScreenContainerProps) {
  const { colors } = useTheme();
  const body = scroll ? (
    <ScrollView
      contentContainerStyle={[padded ? styles.padded : null, styles.grow, contentStyle]}
      keyboardShouldPersistTaps="handled"
      showsVerticalScrollIndicator={false}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.flex, padded ? styles.padded : null, contentStyle]}>{children}</View>
  );
  return (
    <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={[styles.flex, { backgroundColor: colors.background }]}>
      <View style={[styles.flex, styles.page]}>{body}</View>
      {footer ? <View style={[styles.footer, { borderTopColor: colors.border, backgroundColor: colors.background }]}>{footer}</View> : null}
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  // Keeps lines readable on tablets and in the desktop browser.
  page: { width: "100%", maxWidth: 560, alignSelf: "center" },
  grow: { flexGrow: 1 },
  padded: { padding: spacing.xl, gap: spacing.lg },
  footer: { padding: spacing.lg, borderTopWidth: StyleSheet.hairlineWidth, alignItems: "center" }
});
