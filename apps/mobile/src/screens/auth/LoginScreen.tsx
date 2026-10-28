import React from "react";
import { View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import * as authApi from "../../api/auth";
import { useAuth } from "../../auth/AuthContext";
import { AuthForm, Button, Screen, Text } from "../../components";
import { errorMessage } from "../../lib/format";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";

export function LoginScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "Login">) {
  const { signIn } = useAuth();
  const mutation = useMutation({
    mutationFn: ({ email, password }: { email: string; password: string }) => authApi.login(email, password),
    onSuccess: signIn
  });

  return (
    <Screen scroll testID="login-screen">
      <View style={{ gap: spacing.xl, flex: 1, justifyContent: "center" }}>
        <View style={{ gap: spacing.xs }}>
          <Text variant="title">Welcome back</Text>
          <Text tone="muted">Sign in to see who is waiting for you.</Text>
        </View>
        <AuthForm
          submitLabel="Sign in"
          loading={mutation.isPending}
          error={mutation.isError ? errorMessage(mutation.error) : null}
          onSubmit={(email, password) => mutation.mutate({ email, password })}
        />
        <Button title="Forgot your password?" variant="ghost" onPress={() => navigation.navigate("ForgotPassword")} testID="login-forgot" />
        <Button title="Create an account" variant="ghost" onPress={() => navigation.navigate("Register")} />
      </View>
    </Screen>
  );
}
