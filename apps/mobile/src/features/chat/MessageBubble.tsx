import React from "react";
import { StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { formatClock } from "../../shared/format";
import { Text } from "../../shared/ui";
import { radius, spacing, useTheme } from "../../theme";
import type { ThreadRow } from "./threadRows";

type MessageRow = Extract<ThreadRow, { kind: "message" }>;

const STATUS_LABEL = { sending: "Envoi…", sent: "Envoyé", read: "Lu" } as const;

export const MessageBubble = React.memo(function MessageBubble({ row }: { row: MessageRow }) {
  const { colors } = useTheme();
  const { message, mine, status } = row;
  const textColor = mine ? colors.onBubbleMine : colors.text;

  return (
    <View style={[styles.container, mine ? styles.alignEnd : styles.alignStart]}>
      <View
        accessibilityLabel={`${mine ? "Vous" : "Eux"}, ${formatClock(message.createdAt)} : ${message.body}`}
        accessible
        style={[
          styles.bubble,
          mine ? styles.mineShape : styles.theirsShape,
          { backgroundColor: mine ? colors.bubbleMine : colors.bubbleTheirs, opacity: status === "sending" ? 0.7 : 1 }
        ]}
      >
        <Text selectable style={{ color: textColor }}>
          {message.body}
        </Text>
        <Text style={[styles.time, { color: textColor }]} variant="caption">
          {formatClock(message.createdAt)}
        </Text>
      </View>
      {status ? (
        <View style={styles.status}>
          {status === "read" ? <Ionicons color={colors.primary} name="checkmark-done" size={14} /> : null}
          <Text color="textMuted" testID="message-status" variant="caption">
            {STATUS_LABEL[status]}
          </Text>
        </View>
      ) : null}
    </View>
  );
});

export const DaySeparator = React.memo(function DaySeparator({ label }: { label: string }) {
  return (
    <Text color="textMuted" style={styles.separator} variant="caption">
      {label}
    </Text>
  );
});

const styles = StyleSheet.create({
  container: { paddingHorizontal: spacing.lg, paddingVertical: 2, gap: 2 },
  alignEnd: { alignItems: "flex-end" },
  alignStart: { alignItems: "flex-start" },
  bubble: { maxWidth: "80%", borderRadius: radius.md, paddingHorizontal: spacing.md, paddingVertical: spacing.sm },
  mineShape: { borderBottomRightRadius: 4 },
  theirsShape: { borderBottomLeftRadius: 4 },
  time: { alignSelf: "flex-end", opacity: 0.75, fontSize: 11, lineHeight: 14 },
  status: { flexDirection: "row", alignItems: "center", gap: 4 },
  separator: { textAlign: "center", paddingVertical: spacing.md }
});
