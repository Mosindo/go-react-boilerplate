import React from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { Button, Screen, Text } from "../../ui";
import { spacing } from "../../theme/tokens";
import type { AuthStackParams } from "../../navigation/types";

export default function WelcomeScreen({ navigation }: NativeStackScreenProps<AuthStackParams, "Welcome">) {
  return (
    <Screen edges={["top", "bottom"]} testID="welcome-screen">
      <View style={styles.hero}>
        <Text style={styles.mark} accessibilityElementsHidden importantForAccessibility="no">
          ✦
        </Text>
        <Text variant="display" tone="accent">
          Ember
        </Text>
        <Text variant="subtitle" style={styles.tagline}>
          Des rencontres sincères.{"\n"}Gratuites, vraiment.
        </Text>
        <Text tone="muted" style={styles.tagline}>
          Aucun abonnement, aucune limite de likes, aucune option payante.
        </Text>
      </View>
      <View style={styles.actions}>
        <Button
          label="Créer un compte"
          onPress={() => navigation.navigate("Register")}
          testID="welcome-register"
        />
        <Button
          label="J'ai déjà un compte"
          variant="secondary"
          onPress={() => navigation.navigate("Login")}
          testID="welcome-login"
        />
        <Text variant="caption" tone="muted" style={styles.legal}>
          Ember est réservé aux personnes majeures (18 ans et plus).
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md },
  mark: { fontSize: 56, lineHeight: 64 },
  tagline: { textAlign: "center" },
  actions: { gap: spacing.md },
  legal: { textAlign: "center" }
});
