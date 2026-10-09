import React, { useRef, useState } from 'react';
import { StyleSheet, TextInput } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { errorMessage, isApiError } from '../../api/errors';
import { useAuth } from '../../auth/AuthProvider';
import { fr } from '../../i18n/fr';
import type { AuthStackParamList } from '../../navigation/types';
import { Button, Checkbox, Screen, Text, TextField } from '../../shared/ui';
import { spacing } from '../../theme';
import { normalizeEmail, validateEmail, validatePassword } from '../../validators';
import { Brand } from './Brand';

export function RegisterScreen({
  navigation,
}: NativeStackScreenProps<AuthStackParamList, 'Register'>) {
  const { signUp } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [adult, setAdult] = useState(false);
  const [errors, setErrors] = useState<{
    email?: string;
    password?: string;
    adult?: string;
    form?: string;
  }>({});
  const [busy, setBusy] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const submit = async () => {
    const next = {
      email: validateEmail(email) ?? undefined,
      password: validatePassword(password) ?? undefined,
      adult: adult ? undefined : fr.validation.ageConfirm,
    };
    setErrors(next);
    if (next.email || next.password || next.adult) return;
    setBusy(true);
    try {
      await signUp(normalizeEmail(email), password);
    } catch (err) {
      setErrors({ form: isApiError(err, 'conflict') ? fr.auth.emailTaken : errorMessage(err) });
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="register-screen">
      <Brand tagline={false} />
      <Text variant="title">{fr.auth.registerTitle}</Text>
      <Text tone="muted" style={styles.sub}>
        {fr.auth.registerSubtitle}
      </Text>
      <TextField
        testID="register-email"
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
        testID="register-password"
        label={fr.auth.password}
        hint={fr.auth.passwordHint}
        value={password}
        onChangeText={setPassword}
        error={errors.password}
        secure
        autoCapitalize="none"
        autoComplete="new-password"
        textContentType="newPassword"
        returnKeyType="done"
        onSubmitEditing={submit}
      />
      <Checkbox
        testID="register-adult"
        label={fr.auth.ageCheckbox}
        checked={adult}
        onChange={setAdult}
        error={errors.adult}
      />
      <Text variant="caption" tone="muted" style={styles.notice}>
        {fr.auth.ageNotice}
      </Text>
      {errors.form ? (
        <Text tone="danger" accessibilityRole="alert" testID="register-error" style={styles.notice}>
          {errors.form}
        </Text>
      ) : null}
      <Button
        label={fr.auth.createAccount}
        onPress={submit}
        loading={busy}
        testID="register-submit"
      />
      <Button
        label={`${fr.auth.haveAccount} ${fr.auth.login}`}
        variant="ghost"
        onPress={() => navigation.replace('Login')}
        style={styles.link}
        testID="register-to-login"
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  sub: { marginBottom: spacing.xl, marginTop: spacing.xs },
  notice: { marginBottom: spacing.md },
  link: { marginTop: spacing.sm },
});
