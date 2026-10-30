import React, { type ReactNode } from "react";
import {
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
  View,
  type StyleProp,
  type ViewStyle
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { spacing, useTheme } from "./theme";

type ScreenProps = {
  children: ReactNode;
  /** Wrap the content in a ScrollView (forms). Lists should manage their own scrolling. */
  scroll?: boolean;
  /** Apply the top safe-area inset; navigators with a header usually handle it already. */
  topInset?: boolean;
  padded?: boolean;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Screen({
  children,
  scroll = false,
  topInset = false,
  padded = true,
  contentStyle,
  testID
}: ScreenProps) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const inner = [
    styles.inner,
    padded ? styles.padded : null,
    topInset ? { paddingTop: insets.top + spacing.md } : null,
    contentStyle
  ];

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : undefined}
      style={[styles.flex, { backgroundColor: colors.background }]}
      testID={testID}
    >
      {scroll ? (
        <ScrollView
          contentContainerStyle={[inner, styles.scrollContent]}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          {children}
        </ScrollView>
      ) : (
        <View style={inner}>{children}</View>
      )}
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  inner: { flex: 1, width: "100%", maxWidth: 560, alignSelf: "center" },
  padded: { paddingHorizontal: spacing.lg },
  scrollContent: { flexGrow: 1, paddingBottom: spacing.xxl }
});
