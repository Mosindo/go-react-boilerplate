import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { authApi } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import { isValidEmail, isValidPassword, normalizeEmail } from '../../lib/validation';
import type { AuthScreenProps } from '../../navigation/types';
import { Button, Input, SwitchRow, Text, Card } from '../../ui';
import { AuthLayout } from './AuthLayout';
import { useAuth } from './AuthContext';

export function RegisterScreen({ navigation }: AuthScreenProps<'Register'>) {
  const { signIn } = useAuth();
  const onError = useErrorToast();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [adult, setAdult] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const register = useMutation({
    mutationFn: () => authApi.register({ email: normalizeEmail(email), password }),
    onSuccess: signIn,
    onError,
  });

  const emailError = submitted && !isValidEmail(email) ? fr.validation.email : null;
  const passwordError =
    submitted && !isValidPassword(password) ? fr.validation.passwordLength : null;
  const confirmError = submitted && confirm !== password ? fr.validation.passwordMismatch : null;
  const adultError = submitted && !adult;

  const submit = () => {
    setSubmitted(true);
    if (!isValidEmail(email) || !isValidPassword(password) || confirm !== password || !adult)
      return;
    register.mutate();
  };

  return (
    <AuthLayout title={fr.auth.registerTitle} subtitle={fr.auth.registerSubtitle}>
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
        testID="register-email"
      />
      <Input
        label={fr.auth.password}
        helper={fr.auth.passwordHelper}
        value={password}
        onChangeText={setPassword}
        error={passwordError}
        secureTextEntry
        autoCapitalize="none"
        autoComplete="new-password"
        textContentType="newPassword"
        maxLength={72}
        testID="register-password"
      />
      <Input
        label={fr.auth.confirmPassword}
        value={confirm}
        onChangeText={setConfirm}
        error={confirmError}
        secureTextEntry
        autoCapitalize="none"
        autoComplete="new-password"
        textContentType="newPassword"
        maxLength={72}
        testID="register-confirm"
      />
      <Card padded={false}>
        <SwitchRow
          label={fr.auth.adultConfirm}
          value={adult}
          onValueChange={setAdult}
          testID="register-adult"
        />
      </Card>
      {adultError ? (
        <Text variant="caption" color="danger" accessibilityLiveRegion="polite">
          {fr.validation.adultRequired}
        </Text>
      ) : null}
      <Button
        label={fr.auth.registerCta}
        onPress={submit}
        loading={register.isPending}
        testID="register-submit"
      />
      <Button
        label={fr.auth.haveAccount}
        variant="ghost"
        onPress={() => navigation.replace('Login')}
      />
    </AuthLayout>
  );
}
