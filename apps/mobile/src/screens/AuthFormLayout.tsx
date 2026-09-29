import React, { type ReactNode } from "react";
import { View } from "react-native";
import { KeyboardScreen } from "../shared/layout/KeyboardScreen";
import { Button } from "../shared/ui/Button";
import { Text } from "../shared/ui/Text";
import { type Theme } from "../shared/ui/theme";
import { useThemedStyles } from "../shared/ui/useThemedStyles";
import { strings } from "../lib/strings";

type Props = {
  title: string;
  subtitle?: string;
  onBack?: () => void;
  children: ReactNode;
  testID?: string;
};

const makeStyles = (t: Theme) => ({
  brand: { color: t.colors.primary, letterSpacing: 4 },
  header: { gap: t.spacing.xs },
  body: { gap: t.spacing.md },
  back: { alignSelf: "flex-start" as const, marginLeft: -t.spacing.md }
});

export function AuthFormLayout({ children, onBack, subtitle, testID, title }: Props) {
  const styles = useThemedStyles(makeStyles);
  return (
    <KeyboardScreen testID={testID}>
      {onBack ? (
        <Button
          label="Back"
          onPress={onBack}
          size="sm"
          style={styles.back}
          testID="auth-back-button"
          variant="ghost"
        />
      ) : null}
      <View style={styles.header}>
        <Text style={styles.brand} variant="eyebrow" weight="bold">
          {strings.appName}
        </Text>
        <Text accessibilityRole="header" variant="title" weight="bold">
          {title}
        </Text>
        {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
      </View>
      <View style={styles.body}>{children}</View>
    </KeyboardScreen>
  );
}
