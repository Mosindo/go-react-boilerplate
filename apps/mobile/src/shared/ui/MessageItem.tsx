import { StyleSheet, View } from "react-native";
import { formatTime } from "../../lib/dates";
import { radii, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

/** One chat bubble. Text is always rendered as plain text, never as markup. */
export function MessageItem({
  content,
  createdAt,
  mine,
  read,
  failed
}: {
  content: string;
  createdAt: string;
  mine: boolean;
  read?: boolean;
  failed?: boolean;
}) {
  const { colors } = useTheme();
  const time = formatTime(createdAt);
  return (
    <View style={[styles.row, mine ? styles.mine : styles.theirs]}>
      <View
        accessible
        accessibilityLabel={`${mine ? "Vous" : "Message reçu"} à ${time} : ${content}${mine && read ? ". Lu" : ""}`}
        style={[styles.bubble, { backgroundColor: mine ? colors.bubbleMine : colors.bubbleTheirs }]}
      >
        <Text style={{ color: mine ? colors.onBubbleMine : colors.text }}>{content}</Text>
        <Text variant="caption" style={[styles.meta, { color: mine ? colors.onBubbleMine : colors.textMuted }]}>
          {failed ? "Échec de l'envoi" : `${time}${mine ? (read ? " · Lu" : " · Envoyé") : ""}`}
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { paddingHorizontal: spacing.lg, paddingVertical: 2, flexDirection: "row" },
  mine: { justifyContent: "flex-end" },
  theirs: { justifyContent: "flex-start" },
  bubble: { maxWidth: "80%", borderRadius: radii.lg, paddingHorizontal: spacing.md, paddingVertical: spacing.sm },
  meta: { fontSize: 11, lineHeight: 15, opacity: 0.75, alignSelf: "flex-end", marginTop: 2 }
});
