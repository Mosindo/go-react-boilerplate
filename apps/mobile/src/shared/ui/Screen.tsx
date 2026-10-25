import React from "react";
import {
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
  View,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { spacing, useTheme } from "../../theme";

type Props = {
  children: React.ReactNode;
  scroll?: boolean;
  /** Sticky content below the body (primary call to action). */
  footer?: React.ReactNode;
  edges?: Edge[];
  padded?: boolean;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Screen({
  children,
  scroll = false,
  footer,
  edges = ["bottom"],
  padded = true,
  contentStyle,
  testID
}: Props) {
  const { colors } = useTheme();
  const body = scroll ? (
    <ScrollView
      contentContainerStyle={[padded ? styles.padded : null, styles.scrollContent, contentStyle]}
      keyboardDismissMode="on-drag"
      keyboardShouldPersistTaps="handled"
      showsVerticalScrollIndicator={false}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.flex, padded ? styles.padded : null, contentStyle]}>{children}</View>
  );

  return (
    <SafeAreaView edges={edges} style={[styles.flex, { backgroundColor: colors.background }]} testID={testID}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        {body}
        {footer ? <View style={styles.footer}>{footer}</View> : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  padded: { paddingHorizontal: spacing.xl, paddingTop: spacing.lg },
  scrollContent: { paddingBottom: spacing.xxl, gap: spacing.lg },
  footer: { paddingHorizontal: spacing.xl, paddingVertical: spacing.lg, gap: spacing.md }
});
