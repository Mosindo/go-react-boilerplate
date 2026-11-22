import React from "react";
import { StyleSheet, View } from "react-native";
import { Logo } from "../../components/Logo";
import type { AuthScreenProps } from "../../navigation/types";
import { Button, Screen, Text, colors, radii, spacing } from "../../ui";

const promises = [
  "Gratuit pour toujours : likes illimités, aucun abonnement.",
  "Voyez qui vous a liké, sans payer.",
  "Votre position n’est jamais partagée, seulement une distance approximative."
];

export default function WelcomeScreen({ navigation }: AuthScreenProps<"Welcome">) {
  return (
    <Screen
      footer={
        <>
          <Button label="Créer un compte" onPress={() => navigation.navigate("SignUp")} testID="welcome-signup" />
          <Button label="J'ai déjà un compte" onPress={() => navigation.navigate("SignIn")} testID="welcome-signin" variant="secondary" />
        </>
      }
      testID="welcome-screen"
    >
      <View style={styles.hero}>
        <Logo size={44} />
        <Text style={styles.tagline} variant="title">
          Des rencontres sincères,{"\n"}
          <Text tone="accent" variant="title">
            sans rien payer.
          </Text>
        </Text>
        <View style={styles.list}>
          {promises.map((p) => (
            <View key={p} style={styles.item}>
              <View style={styles.dot} />
              <Text style={styles.itemText} tone="muted">
                {p}
              </Text>
            </View>
          ))}
        </View>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: {
    flex: 1,
    justifyContent: "center",
    gap: spacing.xl
  },
  tagline: {
    marginTop: spacing.md
  },
  list: {
    gap: spacing.sm,
    padding: spacing.md,
    borderRadius: radii.lg,
    backgroundColor: colors.surface
  },
  item: {
    flexDirection: "row",
    gap: spacing.sm,
    alignItems: "flex-start"
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: colors.accent,
    marginTop: 8
  },
  itemText: {
    flex: 1
  }
});
