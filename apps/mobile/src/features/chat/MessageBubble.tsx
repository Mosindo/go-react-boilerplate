import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { ChatMessage } from "../../api/types";
import { formatClock } from "../../lib/format";
import { radii, spacing, useColors } from "../../theme";
import { Icon, Text } from "../../ui";

type Props = {
  message: ChatMessage;
  mine: boolean;
  onRetry?: (message: ChatMessage) => void;
  onDiscard?: (message: ChatMessage) => void;
};

/** Short status line under my messages: sending, failed, time and read receipt. */
export function statusLabel(message: ChatMessage): string {
  if (message.status === "sending") return "Sending…";
  if (message.status === "failed") return "Not sent";
  const time = formatClock(message.createdAt);
  return message.readAt ? `${time} · Read` : time;
}

export function MessageBubble({ message, mine, onRetry, onDiscard }: Props) {
  const colors = useColors();
  const failed = message.status === "failed";
  return (
    <View style={[styles.row, mine ? styles.mine : styles.theirs]}>
      <View
        style={[
          styles.bubble,
          {
            backgroundColor: mine ? colors.bubbleMine : colors.bubbleTheirs,
            opacity: message.status === "sending" ? 0.7 : 1
          },
          mine ? styles.bubbleMine : styles.bubbleTheirs
        ]}
      >
        <Text selectable style={{ color: mine ? colors.bubbleMineText : colors.bubbleTheirsText }}>
          {message.body}
        </Text>
      </View>
      {failed ? (
        <View style={styles.failedRow}>
          <Pressable
            accessibilityLabel="Message not sent. Retry"
            accessibilityRole="button"
            hitSlop={8}
            onPress={() => onRetry?.(message)}
            style={styles.failedAction}
          >
            <Icon color={colors.danger} name="refresh" size={16} />
            <Text tone="danger" variant="caption" weight="semibold">
              Not sent. Tap to retry
            </Text>
          </Pressable>
          <Pressable
            accessibilityLabel="Delete unsent message"
            accessibilityRole="button"
            hitSlop={8}
            onPress={() => onDiscard?.(message)}
          >
            <Icon color={colors.textMuted} name="close-circle-outline" size={18} />
          </Pressable>
        </View>
      ) : (
        <Text
          accessibilityLabel={mine ? statusLabel(message).replace("·", ",") : formatClock(message.createdAt)}
          style={styles.status}
          tone="muted"
          variant="caption"
        >
          {mine ? statusLabel(message) : formatClock(message.createdAt)}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { paddingHorizontal: spacing.sm, marginVertical: 3, maxWidth: "100%" },
  mine: { alignItems: "flex-end" },
  theirs: { alignItems: "flex-start" },
  bubble: {
    maxWidth: "82%",
    paddingHorizontal: spacing.sm + 2,
    paddingVertical: spacing.xs + 2,
    borderRadius: radii.lg
  },
  bubbleMine: { borderBottomRightRadius: 6 },
  bubbleTheirs: { borderBottomLeftRadius: 6 },
  status: { marginTop: 2, marginHorizontal: 4 },
  failedRow: { flexDirection: "row", alignItems: "center", gap: spacing.xs, marginTop: 4, minHeight: 28 },
  failedAction: { flexDirection: "row", alignItems: "center", gap: 4 }
});
