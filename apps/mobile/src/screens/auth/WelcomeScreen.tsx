import React from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { AuthStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Screen, Text } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Welcome">;

export function WelcomeScreen({ navigation }: Props) {
  const { colors, radii } = useTheme();
  return (
    <Screen
      testID="welcome-screen"
      footer={
        <View style={styles.actions}>
          <Button testID="welcome-register" label="Créer un compte" onPress={() => navigation.navigate("Register")} />
          <Button
            testID="welcome-login"
            label="J'ai déjà un compte"
            variant="secondary"
            onPress={() => navigation.navigate("Login")}
          />
        </View>
      }
    >
      <View style={styles.hero}>
        <View style={[styles.mark, { backgroundColor: colors.primary, borderRadius: radii.xl }]}>
          <View style={[styles.ring, { borderColor: colors.onPrimary }]} />
          <View style={[styles.ring, styles.ringOffset, { borderColor: colors.onPrimary }]} />
        </View>
        <Text variant="display" align="center">
          Seren
        </Text>
        <Text variant="heading" tone="muted" align="center" style={styles.tagline}>
          Des rencontres sincères, sans abonnement ni paywall.
        </Text>
        <View
          style={[
            styles.promise,
            { backgroundColor: colors.surface, borderColor: colors.border, borderRadius: radii.lg }
          ]}
        >
          {[
            "100 % gratuit, pour toujours",
            "Likes illimités, aucune option payante",
            "Ta position exacte n'est jamais partagée",
            "Blocage et signalement en un geste"
          ].map((line) => (
            <View key={line} style={styles.promiseRow}>
              <View style={[styles.dot, { backgroundColor: colors.primary }]} />
              <Text>{line}</Text>
            </View>
          ))}
        </View>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: 16, paddingVertical: 24 },
  mark: { width: 88, height: 88, alignItems: "center", justifyContent: "center", marginBottom: 8 },
  ring: { position: "absolute", width: 36, height: 36, borderRadius: 18, borderWidth: 3, left: 18, top: 26 },
  ringOffset: { left: 34 },
  tagline: { maxWidth: 320 },
  promise: { marginTop: 16, padding: 18, gap: 12, borderWidth: 1, alignSelf: "stretch" },
  promiseRow: { flexDirection: "row", alignItems: "center", gap: 12 },
  dot: { width: 8, height: 8, borderRadius: 4 },
  actions: { gap: 10 }
});
