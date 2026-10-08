import React from "react";
import { StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import { formatClock } from "../format";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

type MessageItemProps = {
  body: string;
  createdAt: string;
  mine: boolean;
  /** Only meaningful for own messages. */
  read?: boolean;
};

export function MessageItem({ body, createdAt, mine, read = false }: MessageItemProps) {
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
        <Text style={{ color: mine ? colors.onPrimary : colors.text }}>{body}</Text>
        <View style={styles.meta}>
          <Text style={{ color: mine ? colors.onPrimary : colors.textSubtle, opacity: 0.8 }} variant="caption">
            {formatClock(createdAt)}
          </Text>
          {mine ? (
            <Ionicons
              accessibilityLabel={read ? "Lu" : "Envoyé"}
              color={colors.onPrimary}
              name={read ? "checkmark-done" : "checkmark"}
              size={14}
              style={{ opacity: 0.85 }}
            />
          ) : null}
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { paddingHorizontal: spacing.lg, marginVertical: 3, flexDirection: "row" },
  mine: { justifyContent: "flex-end" },
  theirs: { justifyContent: "flex-start" },
  bubble: { maxWidth: "80%", borderRadius: radii.lg, paddingHorizontal: spacing.md, paddingVertical: spacing.sm, gap: 2 },
  meta: { flexDirection: "row", alignSelf: "flex-end", alignItems: "center", gap: 4 }
});
