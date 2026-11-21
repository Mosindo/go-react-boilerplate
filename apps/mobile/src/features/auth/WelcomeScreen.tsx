import React from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../../navigation/types";
import { makeStyles } from "../../theme";
import { BrandMark, Button, Icon, Screen, Text } from "../../ui";

type Props = NativeStackScreenProps<RootStackParamList, "Welcome">;

const promises = [
  { icon: "gift-outline" as const, text: "100 % gratuit : pas d'abonnement, pas de boost, pas de pub." },
  { icon: "heart-outline" as const, text: "Likes illimités, et vous voyez qui vous a liké." },
  { icon: "shield-checkmark-outline" as const, text: "Position approximative, blocage et signalement en un geste." }
];

export function WelcomeScreen({ navigation }: Props) {
  const styles = useStyles();
  return (
    <Screen
      footer={
        <>
          <Button testID="welcome-register" label="Créer un compte" size="lg" onPress={() => navigation.navigate("Register")} />
          <Button testID="welcome-login" label="J'ai déjà un compte" variant="ghost" onPress={() => navigation.navigate("Login")} />
        </>
      }
    >
      <View style={styles.hero}>
        <BrandMark size={88} />
        <Text variant="display" align="center" accessibilityRole="header">
          Lueur
        </Text>
        <Text variant="body" tone="muted" align="center" style={styles.tagline}>
          Des rencontres sincères, près de chez vous. Gratuites, pour de vrai.
        </Text>
      </View>
      <View style={styles.list}>
        {promises.map((p) => (
          <View key={p.text} style={styles.item}>
            <Icon name={p.icon} size={20} color="primary" />
            <Text variant="caption" tone="muted" style={styles.itemText}>
              {p.text}
            </Text>
          </View>
        ))}
      </View>
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  hero: { flex: 1, alignItems: "center", justifyContent: "center", gap: t.spacing.md },
  tagline: { maxWidth: 300 },
  list: { gap: t.spacing.md, paddingBottom: t.spacing.lg },
  item: { flexDirection: "row", alignItems: "center", gap: t.spacing.md },
  itemText: { flex: 1 }
}));
