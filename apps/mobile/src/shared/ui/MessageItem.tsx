import React from "react";
import { StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { radii, spacing } from "./tokens";

type MessageItemProps = {
  body: string;
  time: string;
  mine: boolean;
  /** Only shown on my own messages. */
  status?: "sending" | "sent" | "read" | "failed";
};

const statusLabel = { sending: "Envoi…", sent: "Envoyé", read: "Lu", failed: "Échec de l'envoi" } as const;

export function MessageItem({ body, time, mine, status }: MessageItemProps) {
  const { colors } = useTheme();
  return (
    <View style={[styles.row, mine ? styles.mine : styles.theirs]}>
      <View
        style={[
          styles.bubble,
          mine
            ? { backgroundColor: colors.primary, borderBottomRightRadius: radii.xs }
            : { backgroundColor: colors.surface, borderColor: colors.border, borderWidth: 1, borderBottomLeftRadius: radii.xs }
        ]}
      >
        <Text style={{ color: mine ? colors.primaryForeground : colors.text }}>{body}</Text>
      </View>
      <Text tone={status === "failed" ? "danger" : "subtle"} variant="caption">
        {[time, mine && status ? statusLabel[status] : ""].filter(Boolean).join(" · ")}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { maxWidth: "82%", gap: 3, marginVertical: 3 },
  mine: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { borderRadius: radii.lg, paddingHorizontal: spacing.md, paddingVertical: spacing.sm }
});
