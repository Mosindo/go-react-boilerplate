import { useMutation } from "@tanstack/react-query";
import { login, register } from "../api/auth";
import { normalizeEmail } from "../domain/validation";
import { useAuth } from "./useAuth";

type Credentials = { email: string; password: string };

export function useLogin() {
  const { signIn } = useAuth();
  return useMutation({
    mutationFn: async ({ email, password }: Credentials) => signIn(await login(normalizeEmail(email), password))
  });
}

export function useRegister() {
  const { signIn } = useAuth();
  return useMutation({
    mutationFn: async ({ email, password }: Credentials) => signIn(await register(normalizeEmail(email), password))
  });
}
