import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../../theme";
import { formatClock } from "../../lib/format";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

type MessageItemProps = {
  body: string;
  createdAt: string;
  mine: boolean;
  read?: boolean;
  pending?: boolean;
};

export function MessageItem({ body, createdAt, mine, read, pending }: MessageItemProps) {
  const { colors } = useTheme();
  const status = mine ? (pending ? "Envoi…" : read ? "Lu" : "Envoyé") : null;
  return (
    <View style={[styles.row, mine ? styles.mine : styles.theirs]}>
      <View
        style={[
          styles.bubble,
          mine ? styles.bubbleMine : styles.bubbleTheirs,
          { backgroundColor: mine ? colors.bubbleMine : colors.bubbleTheirs, opacity: pending ? 0.7 : 1 }
        ]}
      >
        <Text selectable style={{ color: mine ? colors.bubbleMineText : colors.text }}>
          {body}
        </Text>
      </View>
      <Text style={styles.meta} tone="subtle" variant="caption">
        {formatClock(createdAt)}
        {status ? ` · ${status}` : ""}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { maxWidth: "82%", gap: 2 },
  mine: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { paddingHorizontal: spacing.md, paddingVertical: spacing.sm, borderRadius: radii.lg },
  bubbleMine: { borderBottomRightRadius: 6 },
  bubbleTheirs: { borderBottomLeftRadius: 6 },
  meta: { paddingHorizontal: spacing.xs }
});
