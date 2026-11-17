import React from "react";
import { StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { RootScreenProps } from "../../navigation/types";
import { colors, spacing } from "../../theme";
import { Button, Screen, Text } from "../../ui";

const promises = [
  { icon: "gift-outline" as const, text: "100 % gratuit. Pas d'abonnement, pas de likes limités." },
  { icon: "shield-checkmark-outline" as const, text: "Ta position exacte n'est jamais partagée." },
  { icon: "hand-left-outline" as const, text: "Bloque ou signale en un geste." }
];

export function WelcomeScreen({ navigation }: RootScreenProps<"Welcome">) {
  return (
    <Screen edges={["top", "bottom", "left", "right"]} testID="welcome-screen">
      <View style={styles.hero}>
        <View style={styles.flame}>
          <Ionicons name="flame" size={40} color={colors.accent} />
        </View>
        <Text variant="display" center>
          Lueur
        </Text>
        <Text variant="heading" tone="muted" center style={styles.tagline}>
          Des rencontres sincères,{"\n"}près de chez toi.
        </Text>
        <View style={styles.promises}>
          {promises.map((p) => (
            <View key={p.text} style={styles.promise}>
              <Ionicons name={p.icon} size={20} color={colors.gold} />
              <Text tone="muted" style={styles.promiseText}>
                {p.text}
              </Text>
            </View>
          ))}
        </View>
      </View>
      <View style={styles.actions}>
        <Button label="Créer un compte" testID="welcome-register" onPress={() => navigation.navigate("Register")} />
        <Button label="J'ai déjà un compte" variant="secondary" testID="welcome-login" onPress={() => navigation.navigate("Login")} />
        <Text variant="caption" tone="subtle" center>
          Réservé aux personnes majeures (18 ans et plus).
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
    gap: spacing.md
  },
  flame: {
    width: 84,
    height: 84,
    borderRadius: 42,
    backgroundColor: colors.accentSoft,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.sm
  },
  tagline: {
    marginBottom: spacing.xl
  },
  promises: {
    gap: spacing.md,
    alignSelf: "stretch"
  },
  promise: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md
  },
  promiseText: {
    flex: 1
  },
  actions: {
    gap: spacing.md,
    paddingBottom: spacing.lg
  }
});
