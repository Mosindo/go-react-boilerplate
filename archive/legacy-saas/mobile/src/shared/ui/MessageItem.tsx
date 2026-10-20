import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { colors, radii, spacing } from "./tokens";

export type MessageItemProps = {
  content: string;
  mine?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function MessageItem({ content, mine = false, style }: MessageItemProps) {
  return (
    <View style={[styles.bubble, mine ? styles.mine : styles.theirs, style]}>
      <Text style={mine ? styles.mineText : styles.theirsText} tone={mine ? "inverse" : "default"}>
        {content}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  bubble: {
    maxWidth: "78%",
    borderRadius: radii.lg,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    marginBottom: spacing.sm
  },
  mine: {
    alignSelf: "flex-end",
    backgroundColor: colors.text
  },
  theirs: {
    alignSelf: "flex-start",
    backgroundColor: colors.backgroundElevated,
    borderWidth: 1,
    borderColor: colors.border
  },
  mineText: {
    color: colors.inverse
  },
  theirsText: {
    color: colors.text
  }
});
