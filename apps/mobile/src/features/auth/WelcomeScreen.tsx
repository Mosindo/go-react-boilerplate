import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../../theme/ThemeProvider";
import { spacing } from "../../theme/palette";
import { Button, Icon, Screen, Text } from "../../ui";
import type { AuthScreenProps } from "../../navigation/types";

export function WelcomeScreen({ navigation }: AuthScreenProps<"Welcome">) {
  const { colors } = useTheme();
  return (
    <Screen edges={["top", "bottom"]} testID="welcome-screen">
      <View style={styles.hero}>
        <View style={[styles.glow, { backgroundColor: colors.primarySoft }]}>
          <Icon name="flame-outline" size={44} color={colors.primary} />
        </View>
        <Text variant="display" align="center" accessibilityRole="header">
          Lueur
        </Text>
        <Text variant="body" tone="muted" align="center" style={styles.tagline}>
          Des rencontres sincères, à deux pas de chez toi. Gratuit, pour de vrai.
        </Text>
      </View>

      <View style={styles.promises}>
        {[
          { icon: "heart-outline" as const, text: "Likes illimités, aucun abonnement" },
          { icon: "shield-checkmark-outline" as const, text: "Position approximative, jamais exacte" },
          { icon: "chatbubbles-outline" as const, text: "On discute seulement si c'est réciproque" }
        ].map((item) => (
          <View key={item.text} style={styles.promise}>
            <Icon name={item.icon} size={20} color={colors.accent} />
            <Text variant="label" tone="muted">
              {item.text}
            </Text>
          </View>
        ))}
      </View>

      <View style={styles.actions}>
        <Button label="Créer un compte" size="lg" fullWidth onPress={() => navigation.navigate("Register")} testID="welcome-register" />
        <Button label="J'ai déjà un compte" variant="outline" size="lg" fullWidth onPress={() => navigation.navigate("Login")} testID="welcome-login" />
        <Text variant="caption" tone="subtle" align="center">
          Réservé aux personnes majeures (18 ans et plus).
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.sm },
  glow: { width: 96, height: 96, borderRadius: 48, alignItems: "center", justifyContent: "center", marginBottom: spacing.sm },
  tagline: { maxWidth: 300 },
  promises: { gap: spacing.sm, marginBottom: spacing.xl, alignSelf: "center" },
  promise: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  actions: { gap: spacing.sm, paddingBottom: spacing.md, width: "100%", maxWidth: 480, alignSelf: "center" }
});
