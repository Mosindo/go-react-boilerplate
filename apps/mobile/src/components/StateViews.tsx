import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { ActivityIndicator, StyleSheet, Text, View } from "react-native";
import { fontSizes, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";
import { Button } from "./Button";

export function LoadingView({ label }: { label?: string }) {
  const theme = useTheme();
  return (
    <View accessibilityLabel={label ?? "Chargement"} style={styles.center}>
      <ActivityIndicator color={theme.primary} size="large" />
      {label ? <Text style={[styles.body, { color: theme.textMuted }]}>{label}</Text> : null}
    </View>
  );
}

type MessageProps = {
  icon: React.ComponentProps<typeof Ionicons>["name"];
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
};

function MessageView({ icon, title, message, actionLabel, onAction }: MessageProps) {
  const theme = useTheme();
  return (
    <View style={styles.center}>
      <View style={[styles.iconWrap, { backgroundColor: theme.primarySoft }]}>
        <Ionicons color={theme.primary} name={icon} size={30} />
      </View>
      <Text style={[styles.title, { color: theme.text }]}>{title}</Text>
      {message ? <Text style={[styles.body, { color: theme.textMuted }]}>{message}</Text> : null}
      {actionLabel && onAction ? (
        <Button onPress={onAction} style={styles.action} title={actionLabel} />
      ) : null}
    </View>
  );
}

export function EmptyView(props: Omit<MessageProps, "icon"> & { icon?: MessageProps["icon"] }) {
  return <MessageView icon="sparkles-outline" {...props} />;
}

export function ErrorView({ message, onRetry }: { message?: string; onRetry?: () => void }) {
  return (
    <MessageView
      actionLabel={onRetry ? "Réessayer" : undefined}
      icon="cloud-offline-outline"
      message={message ?? "Une erreur est survenue. Vérifiez votre connexion et réessayez."}
      onAction={onRetry}
      title="Oups"
    />
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  iconWrap: {
    width: 64,
    height: 64,
    borderRadius: 32,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.lg
  },
  title: { fontSize: fontSizes.lg, fontWeight: "700", textAlign: "center" },
  body: { fontSize: fontSizes.md, textAlign: "center", marginTop: spacing.sm, lineHeight: 22 },
  action: { marginTop: spacing.xl, minWidth: 180 }
});
