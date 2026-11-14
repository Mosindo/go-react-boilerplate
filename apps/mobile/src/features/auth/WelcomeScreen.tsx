import React from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Icon, Screen, Text } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Welcome">;

const promises = [
  { icon: "gift-outline", text: "Entièrement gratuit, sans abonnement ni option payante." },
  { icon: "shield-checkmark-outline", text: "Votre position exacte n'est jamais partagée." },
  { icon: "chatbubbles-outline", text: "On ne discute qu'avec les personnes qui vous plaisent aussi." }
] as const;

export function WelcomeScreen({ navigation }: Props) {
  const { colors } = useTheme();
  return (
    <Screen edges={["top", "bottom", "left", "right"]}>
      <View style={styles.hero}>
        <View style={[styles.mark, { backgroundColor: colors.primary }]}>
          <Icon color={colors.onPrimary} name="flame" size={34} />
        </View>
        <Text accessibilityRole="header" center variant="display">
          Lueur
        </Text>
        <Text center style={styles.tagline} tone="muted">
          Des rencontres sincères, à deux pas de chez vous.
        </Text>
      </View>
      <View style={styles.promises}>
        {promises.map((p) => (
          <View key={p.icon} style={styles.promise}>
            <Icon color={colors.accent} name={p.icon} size={20} />
            <Text style={styles.flex} tone="muted">
              {p.text}
            </Text>
          </View>
        ))}
      </View>
      <View style={styles.actions}>
        <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} size="lg" testID="welcome-register" />
        <Button label="J'ai déjà un compte" onPress={() => navigation.navigate("Login")} variant="ghost" testID="welcome-login" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.sm },
  mark: { width: 76, height: 76, borderRadius: 26, alignItems: "center", justifyContent: "center", marginBottom: spacing.sm },
  tagline: { maxWidth: 300 },
  promises: { gap: spacing.md, marginBottom: spacing.xxl },
  promise: { flexDirection: "row", gap: spacing.sm, alignItems: "flex-start" },
  flex: { flex: 1 },
  actions: { gap: spacing.xs, paddingBottom: spacing.sm }
});
