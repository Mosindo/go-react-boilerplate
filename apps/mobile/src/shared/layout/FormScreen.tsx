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
import { SafeAreaLayout } from "./SafeAreaLayout";
import { spacing } from "../ui/tokens";

type FormScreenProps = {
  children: ReactNode;
  /** Pinned below the scrolling content (e.g. wizard buttons). */
  footer?: ReactNode;
  edges?: ("top" | "right" | "bottom" | "left")[];
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

/** Scrollable, keyboard-aware screen body with the app background and safe areas. */
export function FormScreen({ children, contentStyle, edges, footer, testID }: FormScreenProps) {
  return (
    <SafeAreaLayout edges={edges} testID={testID}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView
          contentContainerStyle={[styles.content, contentStyle]}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          <View style={styles.inner}>{children}</View>
        </ScrollView>
        {footer ? <View style={styles.footer}>{footer}</View> : null}
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { flexGrow: 1, padding: spacing.lg, paddingBottom: spacing.xxl },
  inner: { width: "100%", maxWidth: 560, alignSelf: "center", gap: spacing.lg },
  footer: {
    width: "100%",
    maxWidth: 560,
    alignSelf: "center",
    padding: spacing.lg,
    paddingTop: spacing.sm,
    gap: spacing.sm
  }
});
