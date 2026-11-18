import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { layout, spacing, useTheme } from "../theme";

type Props = {
  children: React.ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  padded?: boolean;
  keyboard?: boolean;
  contentStyle?: ViewStyle;
  testID?: string;
};

export function Screen({ children, scroll, edges = ["top"], padded = true, keyboard, contentStyle, testID }: Props) {
  const { colors } = useTheme();
  const inner = scroll ? (
    <ScrollView
      contentContainerStyle={[styles.scroll, padded ? styles.padded : null, contentStyle]}
      keyboardShouldPersistTaps="handled"
      showsVerticalScrollIndicator={false}
    >
      <View style={styles.column}>{children}</View>
    </ScrollView>
  ) : (
    <View style={[styles.fill, padded ? styles.padded : null, contentStyle]}>{children}</View>
  );

  return (
    <SafeAreaView edges={edges} style={[styles.fill, { backgroundColor: colors.background }]} testID={testID}>
      {keyboard ? (
        <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.fill}>
          {inner}
        </KeyboardAvoidingView>
      ) : (
        inner
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  scroll: { flexGrow: 1, alignItems: "center" },
  padded: { paddingHorizontal: spacing.lg, paddingVertical: spacing.lg },
  column: { width: "100%", maxWidth: layout.maxContentWidth, gap: spacing.lg }
});
