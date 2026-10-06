import React from "react";
import { StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { radii, spacing } from "./tokens";

export type MessageItemProps = {
  body: string;
  time: string;
  mine: boolean;
  /** "Lu" marker shown under my last read message. */
  status?: string;
};

export function MessageItem({ body, time, mine, status }: MessageItemProps) {
  const { colors } = useTheme();
  return (
    <View style={[styles.row, mine ? styles.rowMine : styles.rowTheirs]}>
      <View
        style={[
          styles.bubble,
          mine
            ? { backgroundColor: colors.primary, borderBottomRightRadius: radii.xs }
            : { backgroundColor: colors.surface, borderColor: colors.border, borderWidth: 1, borderBottomLeftRadius: radii.xs }
        ]}
      >
        <Text selectable style={{ color: mine ? colors.primaryForeground : colors.text }}>
          {body}
        </Text>
        <Text style={{ color: mine ? colors.primaryForeground : colors.textSubtle, opacity: 0.8, alignSelf: "flex-end" }} variant="caption">
          {time}
          {status ? ` · ${status}` : ""}
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", paddingHorizontal: spacing.lg, marginVertical: 3 },
  rowMine: { justifyContent: "flex-end" },
  rowTheirs: { justifyContent: "flex-start" },
  bubble: { maxWidth: "80%", borderRadius: radii.lg, paddingHorizontal: spacing.md, paddingVertical: spacing.sm, gap: 2 }
});
