import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { Button } from "../../components/Button";
import { Screen } from "../../components/Screen";
import { Wordmark } from "../../components/Wordmark";
import type { RootStackParamList } from "../../navigation/types";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

type Props = NativeStackScreenProps<RootStackParamList, "Welcome">;

export default function WelcomeScreen({ navigation }: Props) {
  const theme = useTheme();
  return (
    <Screen>
      <View style={styles.hero}>
        <Wordmark size={64} />
        <Text style={[styles.tagline, { color: theme.text }]}>
          Des rencontres qui prennent leur temps.
        </Text>
        <Text style={[styles.sub, { color: theme.textMuted }]}>
          Tout est gratuit : likes illimités, messages illimités, sans abonnement ni publicité.
        </Text>
      </View>
      <View>
        <Button
          onPress={() => navigation.navigate("SignUp")}
          testID="welcome-signup"
          title="Créer un compte"
        />
        <Button
          onPress={() => navigation.navigate("SignIn")}
          style={styles.second}
          testID="welcome-signin"
          title="J'ai déjà un compte"
          variant="secondary"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, justifyContent: "center" },
  tagline: {
    fontSize: 30,
    fontWeight: "700",
    lineHeight: 38,
    marginTop: spacing.xl,
    letterSpacing: -0.5
  },
  sub: { fontSize: fontSizes.md, lineHeight: 24, marginTop: spacing.md },
  second: { marginTop: spacing.md }
});
