import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { spacing, useTheme, type Palette } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type Props = {
  title: string;
  subtitle?: string;
  icon?: IconName;
  tone?: keyof Palette;
  right?: React.ReactNode;
  onPress?: () => void;
  testID?: string;
};

export function ListRow({ title, subtitle, icon, tone = "text", right, onPress, testID }: Props) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceAlt : "transparent" }]}
      testID={testID}
    >
      {icon ? (
        <View style={[styles.iconWrap, { backgroundColor: colors.surfaceAlt }]}>
          <Icon color={tone} name={icon} size={19} />
        </View>
      ) : null}
      <View style={styles.text}>
        <Text color={tone} variant="bodyStrong">
          {title}
        </Text>
        {subtitle ? (
          <Text color="textMuted" variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right ?? (onPress ? <Icon color="textSubtle" name="chevron-forward" size={18} /> : null)}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.xs,
    borderRadius: 14
  },
  iconWrap: { width: 38, height: 38, borderRadius: 19, alignItems: "center", justifyContent: "center" },
  text: { flex: 1, gap: 2 }
});
