import React from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { AuthStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { spacing } from "../../theme/tokens";
import { Button, Icon, Screen, Text } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Welcome">;

export function WelcomeScreen({ navigation }: Props) {
  const { colors } = useTheme();
  return (
    <Screen
      footer={
        <View style={styles.actions}>
          <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
          <Button label="J'ai déjà un compte" variant="secondary" onPress={() => navigation.navigate("Login")} testID="welcome-login" />
        </View>
      }
    >
      <View style={styles.hero}>
        <View style={[styles.logo, { backgroundColor: colors.primary }]}>
          <Icon name="sparkles" size={34} color={colors.onPrimary} />
        </View>
        <Text variant="display" align="center">
          Lueur
        </Text>
        <Text variant="heading" tone="muted" align="center" style={styles.tagline}>
          Des rencontres sincères, près de chez vous.
        </Text>
        <View style={styles.points}>
          {[
            ["heart-outline", "100 % gratuit : likes illimités, aucun abonnement."],
            ["eye-outline", "Voyez qui vous a liké, sans payer."],
            ["shield-checkmark-outline", "Position approximative, blocage et signalement en un geste."]
          ].map(([icon, label]) => (
            <View key={label} style={styles.point}>
              <Icon name={icon as "heart-outline"} size={20} color={colors.accent} />
              <Text tone="muted" style={styles.pointText}>
                {label}
              </Text>
            </View>
          ))}
        </View>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, justifyContent: "center", alignItems: "center", gap: spacing.sm },
  logo: { width: 76, height: 76, borderRadius: 26, alignItems: "center", justifyContent: "center", marginBottom: spacing.sm },
  tagline: { maxWidth: 300, fontWeight: "500" },
  points: { marginTop: spacing.xl, gap: spacing.sm, alignSelf: "stretch" },
  point: { flexDirection: "row", gap: spacing.sm, alignItems: "center" },
  pointText: { flex: 1 },
  actions: { gap: spacing.sm, paddingVertical: spacing.xs }
});
