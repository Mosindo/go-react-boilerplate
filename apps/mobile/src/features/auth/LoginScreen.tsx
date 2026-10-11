import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { authApi } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import { isValidEmail, normalizeEmail } from '../../lib/validation';
import type { AuthScreenProps } from '../../navigation/types';
import { Button, Input } from '../../ui';
import { AuthLayout } from './AuthLayout';
import { useAuth } from './AuthContext';

export function LoginScreen({ navigation }: AuthScreenProps<'Login'>) {
  const { signIn } = useAuth();
  const onError = useErrorToast();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [submitted, setSubmitted] = useState(false);

  const login = useMutation({
    mutationFn: () => authApi.login({ email: normalizeEmail(email), password }),
    onSuccess: signIn,
    onError,
  });

  const emailError = submitted && !isValidEmail(email) ? fr.validation.email : null;
  const passwordError = submitted && password.length === 0 ? fr.validation.passwordRequired : null;

  const submit = () => {
    setSubmitted(true);
    if (!isValidEmail(email) || password.length === 0) return;
    login.mutate();
  };

  return (
    <AuthLayout title={fr.auth.loginTitle} subtitle={fr.auth.loginSubtitle}>
      <Input
        label={fr.auth.email}
        value={email}
        onChangeText={setEmail}
        error={emailError}
        keyboardType="email-address"
        autoCapitalize="none"
        autoCorrect={false}
        autoComplete="email"
        textContentType="username"
        returnKeyType="next"
        testID="login-email"
      />
      <Input
        label={fr.auth.password}
        value={password}
        onChangeText={setPassword}
        error={passwordError}
        secureTextEntry
        autoCapitalize="none"
        autoComplete="current-password"
        textContentType="password"
        returnKeyType="go"
        onSubmitEditing={submit}
        testID="login-password"
      />
      <Button
        label={fr.auth.loginCta}
        onPress={submit}
        loading={login.isPending}
        testID="login-submit"
      />
      <Button
        label={fr.auth.forgotLink}
        variant="ghost"
        onPress={() => navigation.navigate('ForgotPassword')}
      />
    </AuthLayout>
  );
}
