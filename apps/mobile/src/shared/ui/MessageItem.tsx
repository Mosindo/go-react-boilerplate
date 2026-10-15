import React from "react";
import { View } from "react-native";
import { formatClock } from "../../lib/dates";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

export type MessageStatus = "sending" | "sent" | "read" | "failed";

export type MessageItemProps = {
  body: string;
  createdAt: string;
  mine: boolean;
  status?: MessageStatus;
  /** Tightens spacing when it follows a message from the same sender. */
  grouped?: boolean;
};

const statusLabel: Record<MessageStatus, string> = {
  sending: "Envoi…",
  sent: "Envoyé",
  read: "Lu",
  failed: "Échec de l'envoi"
};

export function MessageItem({ body, createdAt, grouped = false, mine, status }: MessageItemProps) {
  const palette = useTheme();
  return (
    <View
      accessible
      accessibilityLabel={`${mine ? "Vous" : "Message reçu"} à ${formatClock(createdAt)} : ${body}`}
      style={{ alignItems: mine ? "flex-end" : "flex-start", marginTop: grouped ? 2 : spacing.md }}
    >
      <View
        style={{
          maxWidth: "82%",
          backgroundColor: mine ? palette.primary : palette.surface,
          borderWidth: mine ? 0 : 1,
          borderColor: palette.border,
          borderRadius: radii.lg,
          borderBottomRightRadius: mine ? radii.xs : radii.lg,
          borderBottomLeftRadius: mine ? radii.lg : radii.xs,
          paddingVertical: spacing.sm + 2,
          paddingHorizontal: spacing.md + 2,
          opacity: status === "sending" ? 0.7 : 1
        }}
      >
        <Text selectable style={{ color: mine ? palette.onPrimary : palette.text }}>
          {body}
        </Text>
      </View>
      <Text style={{ marginTop: 2, marginHorizontal: spacing.xs }} tone={status === "failed" ? "danger" : "subtle"} variant="caption">
        {formatClock(createdAt)}
        {mine && status ? `  ·  ${statusLabel[status]}` : ""}
      </Text>
    </View>
  );
}
