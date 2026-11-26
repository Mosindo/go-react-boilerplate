import React from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { LinearGradient } from "expo-linear-gradient";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { Button, Icon, Screen, Text } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Welcome">;

export function BrandMark({ size = 64 }: { size?: number }) {
  const { colors } = useTheme();
  return (
    <LinearGradient
      colors={[colors.primary, colors.accent]}
      end={{ x: 1, y: 1 }}
      start={{ x: 0, y: 0 }}
      style={{ width: size, height: size, borderRadius: size * 0.32, alignItems: "center", justifyContent: "center" }}
    >
      <Icon color="#FFFFFF" name="flame" size={size * 0.52} />
    </LinearGradient>
  );
}

const promises = [
  { icon: "gift-outline" as const, text: "100 % gratuit. Pas d'abonnement, pas de likes limités." },
  { icon: "shield-checkmark-outline" as const, text: "Vous échangez seulement si l'intérêt est réciproque." },
  { icon: "location-outline" as const, text: "Votre position exacte n'est jamais partagée." }
];

export function WelcomeScreen({ navigation }: Props) {
  const { colors } = useTheme();
  return (
    <Screen
      edges={["top", "bottom"]}
      footer={
        <>
          <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
          <Button label="J'ai déjà un compte" onPress={() => navigation.navigate("Login")} testID="welcome-login" variant="secondary" />
        </>
      }
      testID="welcome-screen"
    >
      <View style={styles.hero}>
        <BrandMark size={84} />
        <Text center variant="display">
          Lueur
        </Text>
        <Text center style={styles.tagline} tone="muted">
          Des rencontres sincères, près de chez vous.
        </Text>
      </View>
      <View style={styles.list}>
        {promises.map((p) => (
          <View key={p.text} style={styles.promise}>
            <View style={[styles.promiseIcon, { backgroundColor: colors.primarySoft }]}>
              <Icon color={colors.primary} name={p.icon} size={20} />
            </View>
            <Text style={styles.promiseText}>{p.text}</Text>
          </View>
        ))}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, paddingTop: spacing.xxl },
  tagline: { maxWidth: 280 },
  list: { gap: spacing.md, paddingBottom: spacing.xl },
  promise: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  promiseIcon: { width: 40, height: 40, borderRadius: 20, alignItems: "center", justifyContent: "center" },
  promiseText: { flex: 1 }
});
