import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  children: React.ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  contentStyle?: ViewStyle;
  footer?: React.ReactNode;
  testID?: string;
};

/** Safe-area aware page with optional scroll, keyboard handling and a sticky footer. */
export function Screen({ children, scroll = true, edges = ["top"], contentStyle, footer, testID }: Props) {
  const { colors, spacing } = useTheme();
  const content = (
    <View style={[styles.content, { padding: spacing.lg, gap: spacing.lg }, contentStyle]}>{children}</View>
  );
  return (
    <SafeAreaView testID={testID} edges={edges} style={[styles.flex, { backgroundColor: colors.background }]}>
      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        {scroll ? (
          <ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={styles.grow}>
            {content}
          </ScrollView>
        ) : (
          content
        )}
        {footer ? (
          <View style={{ paddingHorizontal: spacing.lg, paddingBottom: spacing.lg, paddingTop: spacing.sm }}>
            {footer}
          </View>
        ) : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  grow: { flexGrow: 1 },
  content: { flexGrow: 1, width: "100%", maxWidth: 640, alignSelf: "center" }
});
