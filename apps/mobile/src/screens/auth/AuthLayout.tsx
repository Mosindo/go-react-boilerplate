import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { Screen } from "../../shared/layout";
import { Text, spacing } from "../../shared/ui";

type AuthLayoutProps = { title: string; subtitle?: string; children: ReactNode };

export function AuthLayout({ title, subtitle, children }: AuthLayoutProps) {
  return (
    <Screen scroll>
      <View style={styles.brand}>
        <Text tone="primary" variant="eyebrow" weight="bold">
          Alma
        </Text>
        <Text variant="title" weight="bold">
          {title}
        </Text>
        {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
      </View>
      <View style={styles.form}>{children}</View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  brand: { gap: spacing.xs, marginTop: spacing.xl },
  form: { gap: spacing.md }
});
