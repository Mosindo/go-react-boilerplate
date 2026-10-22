import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { colors, radii, spacing } from "./tokens";

export type MessageItemProps = {
  content: string;
  mine?: boolean;
  /** Small line under the bubble: time, "Envoi…", "Lu"… */
  meta?: string;
  style?: StyleProp<ViewStyle>;
};

export function MessageItem({ content, mine = false, meta, style }: MessageItemProps) {
  return (
    <View style={[styles.wrap, mine ? styles.wrapMine : styles.wrapTheirs, style]}>
      <View style={[styles.bubble, mine ? styles.mine : styles.theirs]}>
        <Text tone={mine ? "inverse" : "default"}>{content}</Text>
      </View>
      {meta ? (
        <Text style={styles.meta} tone="muted" variant="caption">
          {meta}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { maxWidth: "80%", marginBottom: spacing.sm },
  wrapMine: { alignSelf: "flex-end", alignItems: "flex-end" },
  wrapTheirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { borderRadius: radii.lg, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  mine: { backgroundColor: colors.primary, borderBottomRightRadius: radii.xs },
  theirs: { backgroundColor: colors.backgroundElevated, borderWidth: 1, borderColor: colors.border, borderBottomLeftRadius: radii.xs },
  meta: { marginTop: 2, marginHorizontal: spacing.xs }
});
