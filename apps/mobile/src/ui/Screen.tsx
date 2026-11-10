import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";

type ScreenProps = {
  children: ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  padded?: boolean;
  footer?: ReactNode;
  contentStyle?: StyleProp<ViewStyle>;
};

/** Safe-area aware page container with optional scrolling and sticky footer. */
export function Screen({ children, scroll, edges = ["top", "left", "right"], padded = true, footer, contentStyle }: ScreenProps) {
  const theme = useTheme();
  const padding = padded ? { paddingHorizontal: theme.spacing.lg } : null;
  return (
    <SafeAreaView edges={edges} style={[styles.flex, { backgroundColor: theme.colors.background }]}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        {scroll ? (
          <ScrollView
            contentContainerStyle={[styles.scroll, padding, { paddingBottom: theme.spacing.xxl }, contentStyle]}
            keyboardShouldPersistTaps="handled"
            showsVerticalScrollIndicator={false}
          >
            <View style={styles.inner}>{children}</View>
          </ScrollView>
        ) : (
          <View style={[styles.flex, padding, contentStyle]}>{children}</View>
        )}
        {footer ? <View style={[styles.footer, padding, { paddingVertical: theme.spacing.md }]}>{footer}</View> : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  scroll: { flexGrow: 1 },
  inner: { width: "100%", maxWidth: 560, alignSelf: "center", flexGrow: 1 },
  footer: { width: "100%", maxWidth: 600, alignSelf: "center" }
});
