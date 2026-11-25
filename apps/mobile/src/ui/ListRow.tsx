import React from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, spacing } from "../theme";
import { Text } from "./Text";

type ListRowProps = {
  icon: keyof typeof Ionicons.glyphMap;
  title: string;
  subtitle?: string;
  onPress?: () => void;
  danger?: boolean;
  toggle?: { value: boolean; onChange: (v: boolean) => void; disabled?: boolean };
  testID?: string;
};

export function ListRow({ icon, title, subtitle, onPress, danger, toggle, testID }: ListRowProps) {
  const tint = danger ? colors.danger : colors.text;
  const body = (
    <>
      <View style={[styles.icon, danger && styles.iconDanger]}>
        <Ionicons name={icon} size={18} color={danger ? colors.danger : colors.primary} />
      </View>
      <View style={styles.text}>
        <Text variant="bodyStrong" style={{ color: tint }}>
          {title}
        </Text>
        {subtitle ? (
          <Text variant="caption" color="textSubtle">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {toggle ? (
        <Switch
          value={toggle.value}
          onValueChange={toggle.onChange}
          disabled={toggle.disabled}
          trackColor={{ true: colors.primary, false: colors.borderStrong }}
          thumbColor={colors.text}
          accessibilityLabel={title}
          testID={testID}
        />
      ) : onPress ? (
        <Ionicons name="chevron-forward" size={18} color={colors.textSubtle} />
      ) : null}
    </>
  );
  if (!onPress) {
    return <View style={styles.row}>{body}</View>;
  }
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [styles.row, pressed && styles.pressed]}
    >
      {body}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md,
    borderRadius: radii.md,
    backgroundColor: colors.surface
  },
  pressed: {
    backgroundColor: colors.surfaceHigh
  },
  icon: {
    width: 34,
    height: 34,
    borderRadius: 17,
    backgroundColor: colors.primarySoft,
    alignItems: "center",
    justifyContent: "center"
  },
  iconDanger: {
    backgroundColor: colors.dangerSoft
  },
  text: {
    flex: 1,
    gap: 2
  }
});
