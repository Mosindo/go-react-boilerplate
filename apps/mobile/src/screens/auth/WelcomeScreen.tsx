import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { AppText } from "../../components/AppText";
import { Button } from "../../components/Button";
import { Screen } from "../../components/Screen";
import { colors, spacing } from "../../theme/tokens";

const promises: { icon: keyof typeof Ionicons.glyphMap; text: string }[] = [
  { icon: "infinite-outline", text: "Likes illimités, pour tout le monde" },
  { icon: "shield-checkmark-outline", text: "Ta position exacte n'est jamais partagée" },
  { icon: "heart-outline", text: "100 % gratuit : pas d'abonnement, pas de pub" }
];

export default function WelcomeScreen() {
  const navigation = useNavigation();
  return (
    <Screen
      edges={["top", "bottom"]}
      testID="welcome-screen"
      footer={
        <>
          <Button label="Créer un compte" onPress={() => navigation.navigate("SignUp")} testID="welcome-signup" />
          <Button label="J'ai déjà un compte" variant="ghost" onPress={() => navigation.navigate("SignIn")} testID="welcome-signin" />
        </>
      }
    >
      <View style={styles.hero}>
        <View style={styles.glow}>
          <Ionicons name="flame" size={44} color={colors.primary} />
        </View>
        <AppText variant="display" align="center">
          Lueur
        </AppText>
        <AppText variant="heading" color={colors.textMuted} align="center" style={styles.tagline}>
          Des rencontres sincères, sans rien payer.
        </AppText>
        <View style={styles.list}>
          {promises.map((p) => (
            <View key={p.text} style={styles.item}>
              <Ionicons name={p.icon} size={20} color={colors.accent} />
              <AppText color={colors.textMuted} style={styles.itemText}>
                {p.text}
              </AppText>
            </View>
          ))}
        </View>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, paddingHorizontal: spacing.lg },
  glow: {
    width: 96,
    height: 96,
    borderRadius: 48,
    backgroundColor: colors.primarySoft,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.sm
  },
  tagline: { maxWidth: 300 },
  list: { gap: spacing.sm, marginTop: spacing.xl, alignSelf: "stretch" },
  item: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  itemText: { flex: 1 }
});
