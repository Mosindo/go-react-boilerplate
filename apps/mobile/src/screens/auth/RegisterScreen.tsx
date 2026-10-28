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

export function RegisterScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "Register">) {
  const { signIn } = useAuth();
  const mutation = useMutation({
    mutationFn: ({ email, password }: { email: string; password: string }) => authApi.register(email, password),
    onSuccess: signIn
  });

  return (
    <Screen scroll testID="register-screen">
      <View style={{ gap: spacing.xl, flex: 1, justifyContent: "center" }}>
        <View style={{ gap: spacing.xs }}>
          <Text variant="title">Create your account</Text>
          <Text tone="muted">You must be 18 or older. Your profile comes next.</Text>
        </View>
        <AuthForm
          strictPassword
          submitLabel="Continue"
          loading={mutation.isPending}
          error={mutation.isError ? errorMessage(mutation.error) : null}
          onSubmit={(email, password) => mutation.mutate({ email, password })}
        />
        <Button title="I already have an account" variant="ghost" onPress={() => navigation.navigate("Login")} />
      </View>
    </Screen>
  );
}
