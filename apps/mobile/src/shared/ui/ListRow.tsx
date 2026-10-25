import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { MIN_TOUCH_TARGET, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

type Props = {
  title: string;
  subtitle?: string;
  left?: React.ReactNode;
  right?: React.ReactNode;
  onPress?: () => void;
  destructive?: boolean;
  chevron?: boolean;
  accessibilityLabel?: string;
  testID?: string;
};

export const ListRow = React.memo(function ListRow({
  title,
  subtitle,
  left,
  right,
  onPress,
  destructive = false,
  chevron = false,
  accessibilityLabel,
  testID
}: Props) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityLabel={accessibilityLabel ?? (subtitle ? `${title}. ${subtitle}` : title)}
      accessibilityRole={onPress ? "button" : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={({ pressed }) => [styles.row, { opacity: pressed ? 0.6 : 1, borderColor: colors.border }]}
      testID={testID}
    >
      {left}
      <View style={styles.body}>
        <Text color={destructive ? "danger" : "text"} numberOfLines={1} variant="label">
          {title}
        </Text>
        {subtitle ? (
          <Text color="textMuted" numberOfLines={1} variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right}
      {chevron ? <Ionicons color={colors.textMuted} name="chevron-forward" size={18} /> : null}
    </Pressable>
  );
});

const styles = StyleSheet.create({
  row: {
    minHeight: MIN_TOUCH_TARGET + 12,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.sm,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  body: { flex: 1, gap: 2 }
});
