import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Platform, RefreshControl, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaLayout } from "../shared/layout";
import { colors, spacing } from "../shared/ui";

type ScreenProps = {
  children: ReactNode;
  /** Edges that need safe-area padding; stack screens with a native header pass none on top. */
  edges?: ("top" | "right" | "bottom" | "left")[];
  scroll?: boolean;
  refreshing?: boolean;
  onRefresh?: () => void;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Screen({ children, contentStyle, edges = ["right", "left"], onRefresh, refreshing = false, scroll = true, testID }: ScreenProps) {
  const body = scroll ? (
    <ScrollView
      contentContainerStyle={[styles.content, contentStyle]}
      keyboardShouldPersistTaps="handled"
      refreshControl={onRefresh ? <RefreshControl onRefresh={onRefresh} refreshing={refreshing} tintColor={colors.primary} /> : undefined}
      testID={testID}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.content, styles.fill, contentStyle]} testID={testID}>
      {children}
    </View>
  );
  return (
    <SafeAreaLayout edges={edges}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.fill}>
        {body}
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  content: { padding: spacing.lg, gap: spacing.lg, flexGrow: 1 }
});
