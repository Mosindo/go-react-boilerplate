import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { formatClock } from "../lib/dating/format";
import type { ThreadMessageItem } from "../lib/dating/messages";
import { useTheme } from "../shared/ui/theme";
import { Txt } from "./kit";

type Props = {
  item: ThreadMessageItem;
  onRetry?: (localId: string) => void;
  onDiscard?: (localId: string) => void;
};

function MessageBubbleBase({ item, onRetry, onDiscard }: Props) {
  const { colors, radii, spacing } = useTheme();
  const { message, mine, pending, localId, groupedWithOlder } = item;
  const failed = pending === "failed";
  const bg = mine ? (failed ? colors.dangerSoft : colors.primary) : colors.surfaceMuted;
  const fg = mine && !failed ? colors.primaryForeground : colors.text;
  const time = formatClock(message.createdAt);

  const status =
    pending === "sending"
      ? "Sending…"
      : failed
        ? "Not sent. Tap to retry"
        : message.readAt && mine
          ? "Read"
          : null;

  const bubble = (
    <View
      style={[
        styles.bubble,
        {
          backgroundColor: bg,
          borderColor: failed ? colors.danger : "transparent",
          borderRadius: radii.lg,
          paddingHorizontal: spacing.md,
          paddingVertical: spacing.sm,
          opacity: pending === "sending" ? 0.7 : 1
        },
        mine ? { borderBottomRightRadius: 6 } : { borderBottomLeftRadius: 6 }
      ]}
    >
      <Txt style={{ color: fg }} selectable>
        {message.body}
      </Txt>
    </View>
  );

  return (
    <View
      testID={`message-${message.id}`}
      accessible
      accessibilityLabel={`${mine ? "You" : "They"} said: ${message.body}. ${time}${status ? `. ${status}` : ""}`}
      style={[
        styles.row,
        mine ? styles.mine : styles.theirs,
        { marginTop: groupedWithOlder ? 2 : spacing.md }
      ]}
    >
      {failed && localId ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Message not sent. Retry"
          accessibilityHint="Long press to discard"
          onPress={() => onRetry?.(localId)}
          onLongPress={() => onDiscard?.(localId)}
          testID={`message-retry-${localId}`}
          style={styles.maxWidth}
        >
          {bubble}
        </Pressable>
      ) : (
        <View style={styles.maxWidth}>{bubble}</View>
      )}
      <View style={[styles.meta, mine ? styles.metaMine : null]}>
        <Txt variant="caption" tone={failed ? "danger" : "subtle"}>
          {[time, status].filter(Boolean).join(" · ")}
        </Txt>
      </View>
    </View>
  );
}

export const MessageBubble = React.memo(MessageBubbleBase);

const styles = StyleSheet.create({
  row: { paddingHorizontal: 12 },
  mine: { alignItems: "flex-end" },
  theirs: { alignItems: "flex-start" },
  maxWidth: { maxWidth: "80%" },
  bubble: { borderWidth: 1 },
  meta: { marginTop: 2, paddingHorizontal: 4 },
  metaMine: { alignItems: "flex-end" }
});
