import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { spacing, useColors } from "../ui/theme";

type Props = { children: React.ReactNode; scroll?: boolean; padded?: boolean; style?: StyleProp<ViewStyle>; edges?: ("top" | "bottom" | "left" | "right")[] };

export function Screen({ children, scroll = false, padded = true, style, edges = ["top", "left", "right"] }: Props) {
  const c = useColors();
  const body = scroll ? (
    <ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={[padded ? styles.padded : null, styles.grow, style]}>{children}</ScrollView>
  ) : (
    <View style={[styles.flex, padded ? styles.padded : null, style]}>{children}</View>
  );
  return (
    <SafeAreaView edges={edges} style={[styles.flex, { backgroundColor: c.background }]}>
      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}>{body}</KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  grow: { flexGrow: 1 },
  padded: { padding: spacing.lg, gap: spacing.lg },
});
