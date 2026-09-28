import React from "react";
import { StyleSheet, View } from "react-native";
import { BRAND } from "../../theme";
import { spacing, useTheme } from "../../theme";
import { ScreenContainer } from "../../shared/layout";
import { Button, Text } from "../../shared/ui";
import type { AuthScreenProps } from "../../navigation/types";

export default function WelcomeScreen({ navigation }: AuthScreenProps<"Welcome">) {
  const theme = useTheme();
  return (
    <ScreenContainer contentStyle={styles.content}>
      <View style={styles.hero}>
        <View style={[styles.mark, { backgroundColor: theme.primary }]}>
          <View style={[styles.markInner, { backgroundColor: theme.background }]} />
        </View>
        <Text style={styles.name} variant="display">
          {BRAND.name}
        </Text>
        <Text center variant="heading">
          {BRAND.tagline}
        </Text>
        <Text center tone="muted">
          {BRAND.promise} Your location is never shown precisely, and you decide who you talk to.
        </Text>
      </View>
      <View style={styles.actions}>
        <Button label="Create an account" onPress={() => navigation.navigate("Register")} />
        <Button label="I already have an account" onPress={() => navigation.navigate("Login")} variant="secondary" />
        <Text center tone="muted" variant="caption">
          You must be 18 or older to use {BRAND.name}.
        </Text>
      </View>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  content: { flex: 1, justifyContent: "space-between", paddingVertical: spacing.xxl },
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md },
  mark: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center" },
  markInner: { width: 28, height: 28, borderRadius: 14 },
  name: { letterSpacing: 1 },
  actions: { gap: spacing.md }
});
