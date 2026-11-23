import React from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { AuthStackParamList } from "../../navigation/types";
import { Screen } from "../../shared/layout";
import { Button, Text, colors, radii, spacing } from "../../shared/ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Welcome">;

export default function WelcomeScreen({ navigation }: Props) {
  return (
    <Screen
      contentStyle={styles.content}
      edges={["top", "left", "right", "bottom"]}
      footer={
        <>
          <Button fullWidth label="Créer un compte" onPress={() => navigation.navigate("Register")} size="lg" testID="welcome-register" />
          <Button
            fullWidth
            label="J'ai déjà un compte"
            onPress={() => navigation.navigate("Login")}
            testID="welcome-login"
            variant="ghost"
          />
        </>
      }
      scroll={false}
      testID="welcome-screen"
    >
      <View style={styles.hero}>
        <View style={styles.mark}>
          <View style={styles.flame} />
        </View>
        <Text variant="display">Hearth</Text>
        <Text style={styles.tagline} tone="muted" variant="heading" weight="regular">
          Des rencontres sincères, près de chez toi.
        </Text>
      </View>
      <View style={styles.points}>
        {[
          ["Gratuit, pour de vrai", "Pas d'abonnement, pas de likes limités, pas de pub."],
          ["Ta vie privée d'abord", "Ta position exacte n'est jamais partagée."],
          ["Des échanges respectueux", "Bloque ou signale en un geste."]
        ].map(([title, body]) => (
          <View key={title} style={styles.point}>
            <View style={styles.dot} />
            <View style={styles.pointText}>
              <Text weight="semibold">{title}</Text>
              <Text tone="muted" variant="label">
                {body}
              </Text>
            </View>
          </View>
        ))}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: {
    justifyContent: "center",
    gap: spacing.xxxl
  },
  hero: {
    gap: spacing.md
  },
  mark: {
    width: 64,
    height: 64,
    borderRadius: radii.lg,
    backgroundColor: colors.primarySoft,
    borderWidth: 1,
    borderColor: colors.primaryBorder,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.sm
  },
  flame: {
    width: 22,
    height: 30,
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    borderBottomLeftRadius: 11,
    borderBottomRightRadius: 11,
    backgroundColor: colors.primary,
    transform: [{ rotate: "8deg" }]
  },
  tagline: {
    maxWidth: 320
  },
  points: {
    gap: spacing.lg
  },
  point: {
    flexDirection: "row",
    gap: spacing.md
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: colors.accent,
    marginTop: 8
  },
  pointText: {
    flex: 1,
    gap: 2
  }
});
