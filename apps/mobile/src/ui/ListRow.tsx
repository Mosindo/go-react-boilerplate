import React from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { spacing, useTheme, type Palette } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type Props = {
  icon?: IconName;
  label: string;
  description?: string;
  onPress?: () => void;
  tone?: keyof Palette;
  value?: string;
  toggle?: { value: boolean; onChange: (v: boolean) => void; disabled?: boolean };
  testID?: string;
};

export function ListRow({ icon, label, description, onPress, tone = "text", value, toggle, testID }: Props) {
  const { colors } = useTheme();
  const body = (
    <>
      {icon ? (
        <View style={[styles.icon, { backgroundColor: colors.surfaceAlt }]}>
          <Icon name={icon} size={18} color={tone === "text" ? "textMuted" : tone} />
        </View>
      ) : null}
      <View style={styles.texts}>
        <Text variant="label" color={tone} style={styles.label}>
          {label}
        </Text>
        {description ? (
          <Text variant="caption" color="textMuted">
            {description}
          </Text>
        ) : null}
      </View>
      {value ? (
        <Text variant="caption" color="textMuted">
          {value}
        </Text>
      ) : null}
      {toggle ? (
        <Switch
          value={toggle.value}
          onValueChange={toggle.onChange}
          disabled={toggle.disabled}
          trackColor={{ true: colors.primary, false: colors.border }}
          thumbColor="#fff"
          accessibilityLabel={label}
          testID={testID}
        />
      ) : onPress ? (
        <Icon name="chevron-forward" size={18} color="textSubtle" />
      ) : null}
    </>
  );
  if (!onPress || toggle) return <View style={styles.row}>{body}</View>;
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [styles.row, pressed && { opacity: 0.7 }]}
    >
      {body}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingVertical: spacing.sm, minHeight: 56 },
  icon: { width: 36, height: 36, borderRadius: 12, alignItems: "center", justifyContent: "center" },
  texts: { flex: 1, gap: 2 },
  label: { fontSize: 15 }
});
