import React from "react";
import { StyleSheet, View } from "react-native";
import { formatClock } from "../../lib/format";
import { radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

type Props = { body: string; createdAt: string; mine: boolean; read?: boolean; pending?: boolean };

export function MessageItem({ body, createdAt, mine, read, pending }: Props) {
  const theme = useTheme();
  const bg = mine ? theme.primary : theme.surface;
  const fg = mine ? theme.primaryText : theme.text;
  return (
    <View style={[styles.wrap, mine ? styles.mine : styles.theirs]}>
      <View
        style={[
          styles.bubble,
          { backgroundColor: bg, borderColor: mine ? theme.primary : theme.border },
          mine ? styles.bubbleMine : styles.bubbleTheirs,
          pending && { opacity: 0.6 }
        ]}
      >
        <Text style={{ color: fg }}>{body}</Text>
        <Text variant="caption" style={{ color: fg, opacity: 0.75, alignSelf: "flex-end", fontSize: 11 }}>
          {formatClock(createdAt)}
          {mine ? (pending ? " · envoi…" : read ? " · Lu" : "") : ""}
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { paddingHorizontal: spacing.lg, paddingVertical: 3, flexDirection: "row" },
  mine: { justifyContent: "flex-end" },
  theirs: { justifyContent: "flex-start" },
  bubble: { maxWidth: "80%", borderRadius: radius.lg, borderWidth: 1, paddingHorizontal: spacing.md, paddingVertical: spacing.sm, gap: 2 },
  bubbleMine: { borderBottomRightRadius: 6 },
  bubbleTheirs: { borderBottomLeftRadius: 6 }
});
