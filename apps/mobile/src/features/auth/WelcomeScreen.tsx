import React from "react";
import { StyleSheet, View } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { AuthStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme";
import { Button, Screen, Text } from "../../ui";
import { BrandMark } from "./BrandMark";

type Props = NativeStackScreenProps<AuthStackParamList, "Welcome">;

const promises: { icon: keyof typeof Ionicons.glyphMap; text: string }[] = [
  { icon: "gift-outline", text: "100 % gratuit : pas d’abonnement, pas de paywall" },
  { icon: "infinite-outline", text: "Des likes illimités, et tu vois qui t’a liké" },
  { icon: "shield-checkmark-outline", text: "Ta position exacte reste privée" }
];

export function WelcomeScreen({ navigation }: Props) {
  const { colors, spacing } = useTheme();
  return (
    <Screen edges={["top", "bottom", "left", "right"]} contentStyle={{ padding: spacing.xl, justifyContent: "space-between" }}>
      <View style={styles.hero}>
        <BrandMark />
        <Text center tone="muted" variant="body">
          Des rencontres sincères, à ton rythme.
        </Text>
      </View>
      <View style={{ gap: spacing.md }}>
        {promises.map((p) => (
          <View key={p.text} style={styles.promise}>
            <Ionicons color={colors.accent} name={p.icon} size={22} />
            <Text style={styles.flex} tone="muted">
              {p.text}
            </Text>
          </View>
        ))}
      </View>
      <View style={{ gap: spacing.sm }}>
        <Button label="Créer un compte" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
        <Button label="J’ai déjà un compte" onPress={() => navigation.navigate("Login")} testID="welcome-login" variant="ghost" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, justifyContent: "center", gap: 12 },
  promise: { flexDirection: "row", alignItems: "center", gap: 12 },
  flex: { flex: 1 }
});
