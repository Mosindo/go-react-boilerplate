import React, { useRef, useState } from 'react';
import { StyleSheet, TextInput } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { errorMessage, isApiError } from '../../api/errors';
import { useAuth } from '../../auth/AuthProvider';
import { fr } from '../../i18n/fr';
import type { AuthStackParamList } from '../../navigation/types';
import { Button, Screen, Text, TextField } from '../../shared/ui';
import { toast } from '../../shared/toastStore';
import { spacing } from '../../theme';
import { normalizeEmail, validateEmail } from '../../validators';
import { Brand } from './Brand';

export function LoginScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, 'Login'>) {
  const { signIn } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});
  const [busy, setBusy] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const submit = async () => {
    const next = {
      email: validateEmail(email) ?? undefined,
      password: password ? undefined : fr.validation.passwordRequired,
    };
    setErrors(next);
    if (next.email || next.password) return;
    setBusy(true);
    try {
      await signIn(normalizeEmail(email), password);
    } catch (err) {
      const message = isApiError(err, 'unauthorized')
        ? fr.auth.wrongCredentials
        : errorMessage(err);
      setErrors({ form: message });
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="login-screen">
      <Brand tagline={false} />
      <Text variant="title">{fr.auth.loginTitle}</Text>
      <Text tone="muted" style={styles.sub}>
        {fr.auth.loginSubtitle}
      </Text>
      <TextField
        testID="login-email"
        label={fr.auth.email}
        value={email}
        onChangeText={setEmail}
        error={errors.email}
        placeholder={fr.auth.emailPlaceholder}
        keyboardType="email-address"
        autoCapitalize="none"
        autoCorrect={false}
        autoComplete="email"
        textContentType="emailAddress"
        returnKeyType="next"
        onSubmitEditing={() => passwordRef.current?.focus()}
      />
      <TextField
        ref={passwordRef}
        testID="login-password"
        label={fr.auth.password}
        value={password}
        onChangeText={setPassword}
        error={errors.password}
        secure
        autoCapitalize="none"
        autoComplete="current-password"
        textContentType="password"
        returnKeyType="done"
        onSubmitEditing={submit}
      />
      {errors.form ? (
        <Text tone="danger" accessibilityRole="alert" testID="login-error" style={styles.formError}>
          {errors.form}
        </Text>
      ) : null}
      <Button label={fr.auth.login} onPress={submit} loading={busy} testID="login-submit" />
      <Button
        label={fr.auth.forgot}
        variant="ghost"
        onPress={() => {
          toast.clear();
          navigation.navigate('ForgotPassword');
        }}
        style={styles.link}
        testID="login-forgot"
      />
      <Button
        label={`${fr.auth.noAccount} ${fr.auth.createAccount}`}
        variant="ghost"
        onPress={() => navigation.replace('Register')}
        testID="login-to-register"
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  sub: { marginBottom: spacing.xl, marginTop: spacing.xs },
  formError: { marginBottom: spacing.md },
  link: { marginTop: spacing.sm },
});
