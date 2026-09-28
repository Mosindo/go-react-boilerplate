import React, { type ReactNode } from "react";
import { KeyboardAvoidingView, Platform, StyleSheet, View } from "react-native";
import { ScreenContainer } from "../../shared/layout";
import { Text } from "../../shared/ui";
import { spacing } from "../../theme";

export function AuthLayout({ title, subtitle, children }: { title: string; subtitle?: string; children: ReactNode }) {
  return (
    <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
      <ScreenContainer scroll underHeader contentStyle={styles.content}>
        <View style={styles.heading}>
          <Text variant="title">{title}</Text>
          {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
        </View>
        {children}
      </ScreenContainer>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { paddingTop: spacing.xl, gap: spacing.lg },
  heading: { gap: spacing.sm }
});
