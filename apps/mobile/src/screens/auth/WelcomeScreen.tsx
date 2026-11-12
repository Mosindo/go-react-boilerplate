import React from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { Button, Icon, Screen, Text } from "../../ui";

export default function WelcomeScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "Welcome">) {
  const { colors } = useTheme();
  return (
    <Screen edges={["top", "bottom"]}>
      <View style={styles.hero}>
        <View style={[styles.glow, { backgroundColor: colors.primarySoft }]}>
          <Icon name="flame" size={44} color={colors.primary} />
        </View>
        <Text variant="display" serif center accessibilityRole="header">
          Lueur
        </Text>
        <Text variant="body" tone="muted" center style={styles.tagline}>
          Des rencontres sincères, sans abonnement ni like compté. Ici, tout est gratuit, pour tout le monde.
        </Text>
      </View>
      <View style={styles.actions}>
        <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
        <Button label="J'ai déjà un compte" variant="secondary" onPress={() => navigation.navigate("Login")} testID="welcome-login" />
        <Text variant="caption" tone="subtle" center>
          Réservé aux personnes majeures. Vous gardez la main sur vos données.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md },
  glow: { width: 96, height: 96, borderRadius: 48, alignItems: "center", justifyContent: "center", marginBottom: spacing.sm },
  tagline: { maxWidth: 320 },
  actions: { gap: spacing.sm, paddingBottom: spacing.lg }
});
