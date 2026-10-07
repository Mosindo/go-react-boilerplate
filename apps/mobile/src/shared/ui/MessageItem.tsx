import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../../theme/useTheme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

export type MessageItemProps = {
  content: string;
  mine?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function MessageItem({ content, mine = false, style }: MessageItemProps) {
  const { colors } = useTheme();
  return (
    <View
      style={[
        {
          maxWidth: "78%",
          borderRadius: radii.lg,
          paddingHorizontal: spacing.lg,
          paddingVertical: spacing.sm,
          marginBottom: spacing.sm
        },
        mine
          ? { alignSelf: "flex-end", backgroundColor: colors.primary }
          : {
              alignSelf: "flex-start",
              backgroundColor: colors.backgroundElevated,
              borderWidth: 1,
              borderColor: colors.border
            },
        style
      ]}
    >
      <Text style={mine ? { color: colors.primaryForeground } : null} tone={mine ? "inverse" : "default"}>
        {content}
      </Text>
    </View>
  );
}
