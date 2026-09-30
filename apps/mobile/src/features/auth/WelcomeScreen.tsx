import React from "react";
import { StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { Button, Screen, Text } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import type { AuthStackParamList } from "../../navigation/types";

type Props = NativeStackScreenProps<AuthStackParamList, "Welcome">;

const promises = [
  { icon: "gift-outline" as const, text: "100 % gratuit. Pas d'abonnement, pas de likes limités." },
  { icon: "shield-checkmark-outline" as const, text: "Votre position exacte n'est jamais partagée." },
  { icon: "chatbubbles-outline" as const, text: "On ne discute qu'après un intérêt réciproque." }
];

export default function WelcomeScreen({ navigation }: Props) {
  const { colors } = useTheme();
  return (
    <Screen
      footer={
        <View style={styles.actions}>
          <Button fullWidth label="Créer un compte" onPress={() => navigation.navigate("SignUp")} testID="welcome-signup" />
          <Button fullWidth label="J'ai déjà un compte" onPress={() => navigation.navigate("SignIn")} testID="welcome-signin" variant="secondary" />
        </View>
      }
      scroll
      testID="welcome-screen"
    >
      <View style={styles.hero}>
        <View style={[styles.logo, { backgroundColor: colors.primary }]}>
          <Ionicons color={colors.onPrimary} name="flame" size={40} />
        </View>
        <Text align="center" variant="display">
          Lueur
        </Text>
        <Text align="center" style={styles.tagline} tone="muted" variant="heading">
          Des rencontres sincères, près de chez vous.
        </Text>
      </View>
      <View style={styles.list}>
        {promises.map((item) => (
          <View key={item.text} style={styles.promise}>
            <View style={[styles.bullet, { backgroundColor: colors.primarySoft }]}>
              <Ionicons color={colors.primary} name={item.icon} size={20} />
            </View>
            <Text style={styles.promiseText}>{item.text}</Text>
          </View>
        ))}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: 12, marginTop: 48, marginBottom: 24 },
  logo: { width: 84, height: 84, borderRadius: 28, alignItems: "center", justifyContent: "center", marginBottom: 8 },
  tagline: { maxWidth: 300, fontWeight: "400" },
  list: { gap: 16 },
  promise: { flexDirection: "row", alignItems: "center", gap: 14 },
  bullet: { width: 40, height: 40, borderRadius: 20, alignItems: "center", justifyContent: "center" },
  promiseText: { flex: 1 },
  actions: { gap: 12 }
});
