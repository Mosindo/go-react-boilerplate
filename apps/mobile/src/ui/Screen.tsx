import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { useTheme } from "../theme";

type Props = {
  children: React.ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  contentStyle?: StyleProp<ViewStyle>;
  keyboard?: boolean;
  /** Rendered above the scrollable area, so it stays visible and keeps its own padding. */
  header?: React.ReactNode;
};

export function Screen({ children, scroll, edges = ["top", "left", "right"], contentStyle, keyboard, header }: Props) {
  const { colors, spacing } = useTheme();
  const inner = scroll ? (
    <ScrollView
      contentContainerStyle={[{ padding: spacing.lg, gap: spacing.lg, paddingBottom: spacing.xxxl }, contentStyle]}
      keyboardShouldPersistTaps="handled"
      showsVerticalScrollIndicator={false}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.fill, contentStyle]}>{children}</View>
  );
  return (
    <SafeAreaView edges={edges} style={[styles.fill, { backgroundColor: colors.background }]}>
      {header}
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

const styles = StyleSheet.create({ fill: { flex: 1 } });
