import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { spacing, radii, useColors } from "../../theme";
import { Icon, Screen, Text } from "../../ui";

export function Logo() {
  const colors = useColors();
  return (
    <View accessibilityLabel="Lumen" accessibilityRole="image" style={styles.logoRow}>
      <View style={[styles.logoMark, { backgroundColor: colors.primary }]}>
        <Icon color={colors.onPrimary} name="flame" size={22} />
      </View>
      <Text variant="heading">Lumen</Text>
    </View>
  );
}

type AuthLayoutProps = { title: string; subtitle?: string; children: ReactNode };

export function AuthLayout({ title, subtitle, children }: AuthLayoutProps) {
  return (
    <Screen edges={["top", "left", "right", "bottom"]} keyboardAvoiding scroll>
      <View style={styles.header}>
        <Logo />
        <Text variant="display">{title}</Text>
        {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
      </View>
      <View style={styles.form}>{children}</View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  logoRow: { flexDirection: "row", alignItems: "center", gap: spacing.xs, marginBottom: spacing.md },
  logoMark: { width: 40, height: 40, borderRadius: radii.md, alignItems: "center", justifyContent: "center" },
  header: { gap: spacing.xs, paddingTop: spacing.lg },
  form: { gap: spacing.md, paddingTop: spacing.md }
});
