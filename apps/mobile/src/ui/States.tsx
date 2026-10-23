import React from "react";
import { ActivityIndicator, View } from "react-native";
import { t } from "../i18n/fr";
import { createStyles, useTheme } from "../theme/ThemeProvider";
import { Button } from "./Button";
import { FadeIn } from "./FadeIn";
import { Text } from "./Text";

const useStyles = createStyles(({ colors, spacing, radius }) => ({
  wrap: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    padding: spacing.xl,
    gap: spacing.md
  },
  badge: {
    width: 72,
    height: 72,
    borderRadius: radius.lg,
    backgroundColor: colors.surfaceAlt,
    alignItems: "center",
    justifyContent: "center"
  },
  action: { alignSelf: "stretch", maxWidth: 320, width: "100%", marginTop: spacing.sm }
}));

type StateProps = {
  title: string;
  message?: string;
  glyph?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyState({
  title,
  message,
  glyph = "✦",
  actionLabel,
  onAction,
  testID
}: StateProps) {
  const styles = useStyles();
  return (
    <FadeIn style={styles.wrap}>
      <View style={styles.badge} accessibilityElementsHidden importantForAccessibility="no">
        <Text variant="title" tone="accent">
          {glyph}
        </Text>
      </View>
      <View
        testID={testID}
        accessible
        accessibilityLabel={message ? `${title}. ${message}` : title}
      >
        <Text variant="heading" center>
          {title}
        </Text>
        {message ? (
          <Text tone="muted" center>
            {message}
          </Text>
        ) : null}
      </View>
      {actionLabel && onAction ? (
        <View style={styles.action}>
          <Button label={actionLabel} onPress={onAction} variant="secondary" />
        </View>
      ) : null}
    </FadeIn>
  );
}

export function ErrorState({
  title,
  message,
  onRetry
}: {
  title: string;
  message?: string;
  onRetry?: () => void;
}) {
  return (
    <EmptyState
      title={title}
      message={message}
      glyph="!"
      actionLabel={onRetry ? t.common.retry : undefined}
      onAction={onRetry}
    />
  );
}

export function LoadingState() {
  const styles = useStyles();
  const { colors } = useTheme();
  return (
    <View style={styles.wrap} accessibilityRole="progressbar" accessibilityLabel={t.common.loading}>
      <ActivityIndicator size="large" color={colors.primary} />
    </View>
  );
}
