import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type MessageItemProps = {
  content: string;
  mine?: boolean;
  style?: StyleProp<ViewStyle>;
};

const makeStyles = (t: Theme) => ({
  bubble: {
    maxWidth: "78%" as const,
    borderRadius: t.radii.lg,
    paddingHorizontal: t.spacing.lg,
    paddingVertical: t.spacing.sm,
    marginBottom: t.spacing.sm
  },
  mine: { alignSelf: "flex-end" as const, backgroundColor: t.colors.primary },
  theirs: {
    alignSelf: "flex-start" as const,
    backgroundColor: t.colors.surface,
    borderWidth: 1,
    borderColor: t.colors.border
  }
});

export function MessageItem({ content, mine = false, style }: MessageItemProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View style={[styles.bubble, mine ? styles.mine : styles.theirs, style]}>
      <Text tone={mine ? "inverse" : "default"}>{content}</Text>
    </View>
  );
}
