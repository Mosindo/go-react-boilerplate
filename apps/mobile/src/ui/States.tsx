import React from "react";
import { ActivityIndicator, View } from "react-native";
import { makeStyles, useTheme } from "../theme";
import { Button } from "./Button";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export function LoadingState({ label }: { label?: string }) {
  const styles = useStyles();
  const theme = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label ?? "Chargement"}>
      <ActivityIndicator color={theme.colors.primary} size="large" />
      {label ? (
        <Text variant="caption" tone="muted">
          {label}
        </Text>
      ) : null}
    </View>
  );
}

type StateProps = {
  icon?: IconName;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  secondaryLabel?: string;
  onSecondary?: () => void;
};

export function EmptyState({ icon = "sparkles-outline", title, message, actionLabel, onAction, secondaryLabel, onSecondary }: StateProps) {
  const styles = useStyles();
  return (
    <View style={styles.center}>
      <View style={styles.iconWrap}>
        <Icon name={icon} size={30} color="primary" />
      </View>
      <Text variant="title" align="center">
        {title}
      </Text>
      {message ? (
        <Text tone="muted" align="center" style={styles.message}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} /> : null}
      {secondaryLabel && onSecondary ? <Button label={secondaryLabel} variant="ghost" onPress={onSecondary} /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <EmptyState
      icon="cloud-offline-outline"
      title="Oups"
      message={message}
      actionLabel={onRetry ? "Réessayer" : undefined}
      onAction={onRetry}
    />
  );
}

const useStyles = makeStyles((t) => ({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: t.spacing.md, padding: t.spacing.xl },
  iconWrap: {
    width: 64,
    height: 64,
    borderRadius: 32,
    backgroundColor: t.colors.primarySoft,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: t.spacing.sm
  },
  message: { maxWidth: 340, marginBottom: t.spacing.sm }
}));
