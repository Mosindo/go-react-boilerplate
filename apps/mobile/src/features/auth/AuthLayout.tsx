import React from "react";
import { View } from "react-native";
import { Screen } from "../../shared/layout";
import { Text, spacing } from "../../shared/ui";

export function AuthLayout({ title, subtitle, children }: { title: string; subtitle?: string; children: React.ReactNode }) {
  return (
    <Screen scroll style={{ justifyContent: "center" }}>
      <View style={{ gap: spacing.xs, marginBottom: spacing.md }}>
        <Text variant="label" tone="accent" style={{ letterSpacing: 3, textTransform: "uppercase" }}>Liane</Text>
        <Text variant="title">{title}</Text>
        {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
      </View>
      {children}
    </Screen>
  );
}
