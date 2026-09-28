import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { ChatMessage } from "../../api/types";
import { formatClock, receiptGlyph, receiptLabel } from "../../domain/chat";
import { radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export type MessageItemProps = {
  message: ChatMessage;
  mine: boolean;
  showTime: boolean;
  startsGroup: boolean;
  onRetry?: (message: ChatMessage) => void;
};

export function MessageItem({ message, mine, showTime, startsGroup, onRetry }: MessageItemProps) {
  const theme = useTheme();
  const failed = message.status === "failed";
  const bubble = (
    <View
      style={[
        styles.bubble,
        mine
          ? { backgroundColor: failed ? theme.danger : theme.primary, borderBottomRightRadius: radius.sm }
          : {
              backgroundColor: theme.surface,
              borderColor: theme.border,
              borderWidth: StyleSheet.hairlineWidth,
              borderBottomLeftRadius: radius.sm
            },
        message.status === "sending" ? { opacity: 0.7 } : null
      ]}
    >
      <Text style={{ color: mine ? theme.onPrimary : theme.text }}>{message.body}</Text>
    </View>
  );
  const time = formatClock(new Date(message.createdAt));
  return (
    <View style={[styles.row, mine ? styles.mine : styles.theirs, { marginTop: startsGroup ? spacing.md : 2 }]}>
      {failed && onRetry ? (
        <Pressable
          accessibilityHint="Tries to send this message again"
          accessibilityLabel={`Message not sent: ${message.body}. Retry`}
          accessibilityRole="button"
          onPress={() => onRetry(message)}
          style={styles.retryTarget}
        >
          {bubble}
        </Pressable>
      ) : (
        <View
          accessible
          accessibilityLabel={`${mine ? "You" : "Them"}: ${message.body}. ${time}${mine ? `. ${receiptLabel(message)}` : ""}`}
        >
          {bubble}
        </View>
      )}
      {showTime ? (
        <View style={[styles.meta, mine ? { alignSelf: "flex-end" } : null]}>
          <Text tone="muted" variant="caption">
            {time}
          </Text>
          {mine ? (
            <Text
              style={{ color: failed ? theme.danger : message.readAt ? theme.primary : theme.textMuted }}
              variant="caption"
            >
              {receiptGlyph(message)}
            </Text>
          ) : null}
          {failed ? (
            <Text tone="danger" variant="caption">
              Not sent. Tap to retry
            </Text>
          ) : null}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { paddingHorizontal: spacing.lg, maxWidth: "100%" },
  mine: { alignItems: "flex-end" },
  theirs: { alignItems: "flex-start" },
  bubble: { maxWidth: "82%", borderRadius: radius.lg, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm + 2 },
  retryTarget: { maxWidth: "100%", minHeight: 44, justifyContent: "center" },
  meta: { flexDirection: "row", gap: spacing.xs, marginTop: 2, alignItems: "center" }
});
