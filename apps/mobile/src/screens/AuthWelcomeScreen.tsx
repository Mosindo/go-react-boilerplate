import React from "react";
import { View } from "react-native";
import { strings } from "../lib/strings";
import { SafeAreaLayout } from "../shared/layout/SafeAreaLayout";
import { Button } from "../shared/ui/Button";
import { Notice } from "../shared/ui/Notice";
import { Text } from "../shared/ui/Text";
import { type Theme } from "../shared/ui/theme";
import { useThemedStyles } from "../shared/ui/useThemedStyles";

type Props = {
  notice: string | null;
  onLogin: () => void;
  onRegister: () => void;
};

const makeStyles = (t: Theme) => ({
  root: { flex: 1, padding: t.spacing.xl, justifyContent: "space-between" as const, maxWidth: 560, width: "100%" as const, alignSelf: "center" as const },
  hero: { flex: 1, justifyContent: "center" as const, gap: t.spacing.md },
  logo: { color: t.colors.primary, fontSize: 56, lineHeight: 64, letterSpacing: -1 },
  actions: { gap: t.spacing.sm },
  promise: { textAlign: "center" as const }
});

export function AuthWelcomeScreen({ notice, onLogin, onRegister }: Props) {
  const styles = useThemedStyles(makeStyles);
  return (
    <SafeAreaLayout>
      <View style={styles.root} testID="welcome-screen">
        <View style={styles.hero}>
          <Text accessibilityRole="header" style={styles.logo} weight="bold">
            {strings.appName}
          </Text>
          <Text variant="heading" weight="bold">
            {strings.auth.welcomeTitle}
          </Text>
          <Text tone="muted">{strings.auth.welcomeBody}</Text>
          {notice ? <Notice title={notice} tone="default" /> : null}
        </View>
        <View style={styles.actions}>
          <Button
            label={strings.auth.createAccount}
            onPress={onRegister}
            size="lg"
            testID="welcome-register-button"
          />
          <Button
            label={strings.auth.signIn}
            onPress={onLogin}
            size="lg"
            testID="welcome-login-button"
            variant="outline"
          />
          <Text style={styles.promise} tone="muted" variant="caption">
            {strings.freePromise}
          </Text>
        </View>
      </View>
    </SafeAreaLayout>
  );
}
