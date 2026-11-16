import React from "react";
import { StyleSheet, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { AuthScreenProps } from "../../navigation/types";
import { colors, spacing } from "../../theme/tokens";
import { Button, Screen, Text } from "../../ui";

export default function WelcomeScreen({ navigation }: AuthScreenProps<"Welcome">) {
  return (
    <Screen edges={["top", "bottom"]} testID="welcome-screen">
      <LinearGradient colors={["rgba(240,138,93,0.22)", "transparent"]} style={StyleSheet.absoluteFill} pointerEvents="none" />
      <View style={styles.hero}>
        <View style={styles.glow} />
        <Text variant="display" style={styles.brand} align="center">
          Lueur
        </Text>
        <Text variant="title" align="center" style={styles.tagline}>
          Des rencontres sincères, entièrement gratuites.
        </Text>
        <Text tone="muted" align="center">
          Pas d&apos;abonnement, pas de likes limités, pas de fonctionnalité payante. Juste des gens.
        </Text>
      </View>
      <View style={styles.actions}>
        <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
        <Button label="J'ai déjà un compte" variant="secondary" onPress={() => navigation.navigate("Login")} testID="welcome-login" />
        <Text variant="caption" tone="subtle" align="center">
          Réservé aux personnes majeures (18 ans et plus).
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, justifyContent: "center", gap: spacing.lg, paddingHorizontal: spacing.md },
  glow: { alignSelf: "center", width: 72, height: 72, borderRadius: 36, backgroundColor: colors.accent, opacity: 0.9, marginBottom: spacing.md },
  brand: { fontSize: 52, lineHeight: 58, letterSpacing: 1 },
  tagline: { color: colors.text },
  actions: { gap: spacing.md, paddingBottom: spacing.lg }
});
