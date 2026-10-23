import React from "react";
import { View } from "react-native";
import { t } from "../../i18n/fr";
import { createStyles } from "../../theme/ThemeProvider";
import { FadeIn, Screen, Text } from "../../ui";

const useStyles = createStyles(({ colors, spacing }) => ({
  brand: { alignItems: "center", gap: spacing.sm, paddingTop: spacing.xl },
  mark: {
    width: 64,
    height: 64,
    borderRadius: 32,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center",
    borderBottomRightRadius: 12
  },
  form: { gap: spacing.lg }
}));

export function AuthLayout({
  title,
  subtitle,
  children,
  testID
}: {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
  testID?: string;
}) {
  const styles = useStyles();
  return (
    <Screen scroll edges={["top", "bottom"]} testID={testID}>
      <FadeIn style={styles.brand}>
        <View accessibilityElementsHidden importantForAccessibility="no" style={styles.mark}>
          <Text variant="title" tone="onPrimary">
            C
          </Text>
        </View>
        <Text variant="heading" tone="primary">
          {t.appName}
        </Text>
      </FadeIn>
      <FadeIn delay={80}>
        <Text variant="title" center>
          {title}
        </Text>
        {subtitle ? (
          <Text tone="muted" center>
            {subtitle}
          </Text>
        ) : null}
      </FadeIn>
      <FadeIn delay={140} style={styles.form}>
        {children}
      </FadeIn>
    </Screen>
  );
}
