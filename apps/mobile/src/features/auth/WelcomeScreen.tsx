import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { StyleSheet, View } from "react-native";
import type { RootScreenProps } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Screen, Text } from "../../ui";

const promises: { icon: keyof typeof Ionicons.glyphMap; text: string }[] = [
  { icon: "heart-outline", text: "100 % gratuit : pas d'abonnement, pas de likes limités, pas de pub." },
  { icon: "shield-checkmark-outline", text: "Votre position exacte n'est jamais partagée." },
  { icon: "hand-left-outline", text: "Bloquez ou signalez en un geste, à tout moment." }
];

export default function WelcomeScreen({ navigation }: RootScreenProps<"Welcome">) {
  const { colors, spacing } = useTheme();
  return (
    <Screen
      testID="welcome-screen"
      footer={
        <View style={{ gap: spacing.sm }}>
          <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
          <Button
            label="J'ai déjà un compte"
            variant="secondary"
            onPress={() => navigation.navigate("Login")}
            testID="welcome-login"
          />
        </View>
      }
    >
      <View style={styles.hero}>
        <View style={[styles.mark, { backgroundColor: colors.primary }]}>
          <Ionicons name="magnet" size={34} color={colors.onPrimary} />
        </View>
        <Text variant="display" align="center" accessibilityRole="header">
          Aimant
        </Text>
        <Text variant="heading" weight="regular" tone="muted" align="center" style={styles.tagline}>
          Des rencontres sincères, près de chez vous.
        </Text>
      </View>
      <View style={{ gap: spacing.md, marginBottom: spacing.xl }}>
        {promises.map((p) => (
          <View key={p.text} style={styles.promise}>
            <Ionicons name={p.icon} size={22} color={colors.accent} />
            <Text tone="muted" style={styles.flex}>
              {p.text}
            </Text>
          </View>
        ))}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12 },
  mark: { width: 76, height: 76, borderRadius: 38, alignItems: "center", justifyContent: "center", marginBottom: 8 },
  tagline: { maxWidth: 300 },
  promise: { flexDirection: "row", gap: 12, alignItems: "flex-start" },
  flex: { flex: 1 }
});
