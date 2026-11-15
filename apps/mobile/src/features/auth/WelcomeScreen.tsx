import React from "react";
import { StyleSheet, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { AuthStackParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme";
import { Button, Icon, Screen, Text, type IconName } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Welcome">;

const promises: { icon: IconName; text: string }[] = [
  { icon: "heart-outline", text: "Likes illimités, pour tout le monde" },
  { icon: "sparkles-outline", text: "Aucun abonnement, aucune option payante" },
  { icon: "shield-checkmark-outline", text: "Blocage, signalement et position approximative" }
];

export function WelcomeScreen({ navigation }: Props) {
  const { colors } = useTheme();
  return (
    <Screen testID="welcome-screen" edges={["top", "bottom"]}>
      <LinearGradient
        colors={[colors.primarySoft, colors.background]}
        style={StyleSheet.absoluteFill}
        start={{ x: 0.2, y: 0 }}
        end={{ x: 0.8, y: 0.7 }}
      />
      <View style={styles.hero}>
        <View style={[styles.mark, { backgroundColor: colors.primary }]}>
          <Icon name="flame" size={34} color="onPrimary" />
        </View>
        <Text variant="display" align="center" accessibilityRole="header">
          Lueur
        </Text>
        <Text variant="body" color="textMuted" align="center" style={styles.tagline}>
          Des rencontres sincères, à deux pas de chez vous. Entièrement gratuites.
        </Text>
      </View>
      <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
        {promises.map((p) => (
          <View key={p.text} style={styles.promise}>
            <Icon name={p.icon} size={20} color="primary" />
            <Text variant="label" style={styles.promiseText}>
              {p.text}
            </Text>
          </View>
        ))}
      </View>
      <View style={styles.actions}>
        <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
        <Button label="J'ai déjà un compte" variant="secondary" onPress={() => navigation.navigate("Login")} testID="welcome-login" />
        <Text variant="caption" color="textSubtle" align="center">
          Réservé aux personnes majeures (18 ans et plus).
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, paddingTop: spacing.xl },
  mark: { width: 72, height: 72, borderRadius: 24, alignItems: "center", justifyContent: "center", transform: [{ rotate: "-8deg" }] },
  tagline: { maxWidth: 300 },
  card: { borderRadius: radii.lg, borderWidth: 1, padding: spacing.md, gap: spacing.sm, marginBottom: spacing.lg },
  promise: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  promiseText: { flex: 1 },
  actions: { gap: spacing.sm, paddingBottom: spacing.md }
});
