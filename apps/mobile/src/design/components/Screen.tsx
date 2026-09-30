import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { useTheme } from "../ThemeProvider";

type Props = {
  children: React.ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  contentStyle?: StyleProp<ViewStyle>;
  footer?: React.ReactNode;
  testID?: string;
};

/** Page shell: safe areas, keyboard avoidance, max width on large screens. */
export function Screen({ children, scroll, edges = ["top"], contentStyle, footer, testID }: Props) {
  const { colors, spacing } = useTheme();
  const body = scroll ? (
    <ScrollView
      contentContainerStyle={[styles.scrollContent, { padding: spacing.lg }, contentStyle]}
      keyboardShouldPersistTaps="handled"
      showsVerticalScrollIndicator={false}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.fill, contentStyle]}>{children}</View>
  );
  return (
    <SafeAreaView edges={edges} style={[styles.fill, { backgroundColor: colors.background }]} testID={testID}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.fill}>
        <View style={styles.column}>{body}</View>
        {footer ? <View style={[styles.column, styles.footer, { padding: spacing.lg }]}>{footer}</View> : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  column: { flex: 1, width: "100%", maxWidth: 640, alignSelf: "center" },
  footer: { flex: 0 },
  scrollContent: { flexGrow: 1, gap: 16 }
});
