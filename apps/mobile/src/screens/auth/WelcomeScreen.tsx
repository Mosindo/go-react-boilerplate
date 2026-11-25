import React from "react";
import { StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { AuthScreenProps } from "../../navigation/types";
import { colors, spacing } from "../../theme";
import { Button, Screen, Text } from "../../ui";

const promises: { icon: keyof typeof Ionicons.glyphMap; text: string }[] = [
  { icon: "heart-outline", text: "100 % gratuit, sans abonnement ni likes limités." },
  { icon: "shield-checkmark-outline", text: "Votre position exacte n'est jamais partagée." },
  { icon: "chatbubbles-outline", text: "On ne discute qu'après un intérêt réciproque." }
];

export function WelcomeScreen({ navigation }: AuthScreenProps<"Welcome">) {
  return (
    <Screen
      testID="welcome-screen"
      footer={
        <>
          <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
          <Button label="J'ai déjà un compte" variant="secondary" onPress={() => navigation.navigate("Login")} testID="welcome-login" />
        </>
      }
    >
      <View style={styles.hero}>
        <View style={styles.glow}>
          <Ionicons name="flame" size={44} color={colors.primary} />
        </View>
        <Text variant="display" align="center">
          Lueur
        </Text>
        <Text variant="body" color="textMuted" align="center" style={styles.tagline}>
          Des rencontres sincères, près de chez vous.
        </Text>
      </View>
      <View style={styles.promises}>
        {promises.map((p) => (
          <View key={p.text} style={styles.promise}>
            <Ionicons name={p.icon} size={20} color={colors.gold} />
            <Text variant="small" color="textMuted" style={styles.flex}>
              {p.text}
            </Text>
          </View>
        ))}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.sm
  },
  glow: {
    width: 96,
    height: 96,
    borderRadius: 48,
    backgroundColor: colors.primarySoft,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.sm
  },
  tagline: {
    maxWidth: 280
  },
  promises: {
    gap: spacing.sm,
    paddingBottom: spacing.lg
  },
  promise: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  },
  flex: {
    flex: 1
  }
});
