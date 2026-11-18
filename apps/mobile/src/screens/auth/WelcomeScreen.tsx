import React from "react";
import { StyleSheet, View } from "react-native";
import type { AuthScreenProps } from "../../navigation/types";
import { makeStyles, spacing } from "../../theme";
import { Button, Icon, Screen, Text } from "../../ui";

export default function WelcomeScreen({ navigation }: AuthScreenProps<"Welcome">) {
  const styles = useStyles();
  return (
    <Screen edges={["top", "bottom"]} testID="welcome-screen">
      <View style={styles.hero}>
        <View style={styles.mark}>
          <Icon color="textOnAccent" name="flame" size={34} />
        </View>
        <Text align="center" style={styles.brand} variant="display">
          Lueur
        </Text>
        <Text align="center" color="textMuted" style={styles.tagline}>
          Des rencontres sincères, près de chez vous. Gratuites, pour de vrai.
        </Text>
        <View style={styles.pills}>
          {["Sans abonnement", "Sans publicité", "Likes illimités"].map((label) => (
            <View key={label} style={styles.pill}>
              <Icon color="accent" name="checkmark" size={14} />
              <Text color="textMuted" variant="caption">
                {label}
              </Text>
            </View>
          ))}
        </View>
      </View>
      <View style={styles.actions}>
        <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
        <Button label="J’ai déjà un compte" onPress={() => navigation.navigate("Login")} testID="welcome-login" variant="secondary" />
        <Text align="center" color="textSubtle" variant="caption">
          Réservé aux personnes majeures.
        </Text>
      </View>
    </Screen>
  );
}

const useStyles = makeStyles((c) => ({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md },
  mark: {
    width: 76,
    height: 76,
    borderRadius: 38,
    backgroundColor: c.accent,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.sm
  },
  brand: { fontSize: 52, lineHeight: 58 },
  tagline: { maxWidth: 320 },
  pills: { flexDirection: "row", flexWrap: "wrap", justifyContent: "center", gap: spacing.xs, marginTop: spacing.sm },
  pill: {
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    paddingHorizontal: spacing.sm,
    paddingVertical: 6,
    borderRadius: 999,
    backgroundColor: c.surface,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: c.border
  },
  actions: { gap: spacing.sm, width: "100%", maxWidth: 480, alignSelf: "center" }
}));
