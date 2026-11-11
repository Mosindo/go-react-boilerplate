import React, { useState } from "react";
import { Pressable, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import type { AuthStackParamList } from "../../navigation/types";
import { Button, Screen, Text, TextField } from "../../ui";

type Props = NativeStackScreenProps<AuthStackParamList, "Login">;

export function LoginScreen({ navigation }: Props) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const login = useMutation({
    mutationFn: () => authApi.login(email.trim(), password),
    onSuccess: signIn
  });

  const canSubmit = email.trim().length > 3 && password.length > 0;

  return (
    <Screen
      testID="login-screen"
      footer={
        <Button
          testID="login-submit"
          label="Se connecter"
          loading={login.isPending}
          disabled={!canSubmit}
          onPress={() => login.mutate()}
        />
      }
    >
      <View style={{ gap: 6 }}>
        <Text variant="title">Bon retour</Text>
        <Text tone="muted">Connecte-toi pour retrouver tes matchs.</Text>
      </View>
      <TextField
        testID="login-email"
        label="E-mail"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        textContentType="emailAddress"
      />
      <TextField
        testID="login-password"
        label="Mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoComplete="password"
        textContentType="password"
        onSubmitEditing={() => canSubmit && login.mutate()}
      />
      {login.isError ? (
        <Text tone="danger" accessibilityLiveRegion="polite">
          {errorMessage(login.error)}
        </Text>
      ) : null}
      <Pressable
        accessibilityRole="link"
        onPress={() => navigation.navigate("ForgotPassword", { email: email.trim() })}
      >
        <Text tone="primary" variant="label">
          Mot de passe oublié ?
        </Text>
      </Pressable>
    </Screen>
  );
}
