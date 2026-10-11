import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { authApi } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import { isValidEmail, normalizeEmail } from '../../lib/validation';
import type { AuthScreenProps } from '../../navigation/types';
import { Button, Card, Input, Text } from '../../ui';
import { AuthLayout } from './AuthLayout';

export function ForgotPasswordScreen({ navigation }: AuthScreenProps<'ForgotPassword'>) {
  const onError = useErrorToast();
  const [email, setEmail] = useState('');
  const [submitted, setSubmitted] = useState(false);
  const [sent, setSent] = useState(false);

  const request = useMutation({
    mutationFn: () => authApi.forgotPassword(normalizeEmail(email)),
    onSuccess: () => setSent(true),
    onError,
  });

  const emailError = submitted && !isValidEmail(email) ? fr.validation.email : null;
  const submit = () => {
    setSubmitted(true);
    if (!isValidEmail(email)) return;
    request.mutate();
  };

  return (
    <AuthLayout title={fr.auth.forgotTitle} subtitle={fr.auth.forgotSubtitle}>
      <Input
        label={fr.auth.email}
        value={email}
        onChangeText={setEmail}
        error={emailError}
        keyboardType="email-address"
        autoCapitalize="none"
        autoCorrect={false}
        autoComplete="email"
        returnKeyType="send"
        onSubmitEditing={submit}
      />
      {sent ? (
        <Card accessibilityLiveRegion="polite">
          <Text color="success" weight="600">
            {fr.auth.forgotSent}
          </Text>
        </Card>
      ) : null}
      <Button
        label={sent ? fr.auth.forgotResend : fr.auth.forgotCta}
        onPress={submit}
        loading={request.isPending}
        variant={sent ? 'ghost' : 'primary'}
      />
      <Button
        label={fr.auth.haveCode}
        variant={sent ? 'primary' : 'ghost'}
        onPress={() => navigation.navigate('ResetPassword')}
      />
    </AuthLayout>
  );
}
