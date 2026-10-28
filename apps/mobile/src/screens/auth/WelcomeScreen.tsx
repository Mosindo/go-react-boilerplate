import React from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { Button, Screen, Text } from "../../components";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";

export function WelcomeScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "Welcome">) {
  const { colors } = useTheme();
  return (
    <Screen testID="welcome-screen">
      <View style={{ flex: 1, justifyContent: "center", gap: spacing.md }}>
        <View style={{ width: 64, height: 64, borderRadius: 32, backgroundColor: colors.accent, marginBottom: spacing.lg }} />
        <Text variant="title" style={{ fontSize: 44, lineHeight: 50 }}>
          Lumen
        </Text>
        <Text tone="muted" style={{ fontSize: 18, lineHeight: 26 }}>
          Meet people you actually click with. Completely free: no subscriptions, no limits, no paywalls.
        </Text>
      </View>
      <View style={{ gap: spacing.sm }}>
        <Button title="Create an account" onPress={() => navigation.navigate("Register")} testID="welcome-register" />
        <Button title="I already have an account" variant="secondary" onPress={() => navigation.navigate("Login")} testID="welcome-login" />
      </View>
    </Screen>
  );
}
