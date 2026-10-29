import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Pressable, StyleSheet, Switch, Text, View } from "react-native";
import { fontSizes, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  title: string;
  subtitle?: string;
  icon?: React.ComponentProps<typeof Ionicons>["name"];
  onPress?: () => void;
  destructive?: boolean;
  /** Renders a switch instead of a chevron. */
  switchValue?: boolean;
  onSwitchChange?: (value: boolean) => void;
};

export function ListRow({
  title,
  subtitle,
  icon,
  onPress,
  destructive,
  switchValue,
  onSwitchChange
}: Props) {
  const theme = useTheme();
  const color = destructive ? theme.danger : theme.text;
  const isSwitch = switchValue !== undefined && onSwitchChange !== undefined;
  return (
    <Pressable
      accessibilityRole={isSwitch ? "switch" : "button"}
      accessibilityState={isSwitch ? { checked: switchValue } : undefined}
      disabled={!onPress && !isSwitch}
      onPress={isSwitch ? () => onSwitchChange(!switchValue) : onPress}
      style={({ pressed }) => [
        styles.row,
        {
          borderBottomColor: theme.border,
          backgroundColor: pressed ? theme.surfaceMuted : "transparent"
        }
      ]}
    >
      {icon ? <Ionicons color={color} name={icon} size={22} style={styles.icon} /> : null}
      <View style={styles.text}>
        <Text style={[styles.title, { color }]}>{title}</Text>
        {subtitle ? (
          <Text style={[styles.subtitle, { color: theme.textMuted }]}>{subtitle}</Text>
        ) : null}
      </View>
      {isSwitch ? (
        <Switch
          onValueChange={onSwitchChange}
          trackColor={{ true: theme.primary, false: theme.border }}
          value={switchValue}
        />
      ) : onPress ? (
        <Ionicons color={theme.textMuted} name="chevron-forward" size={18} />
      ) : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.lg,
    borderBottomWidth: StyleSheet.hairlineWidth,
    minHeight: 56
  },
  icon: { marginRight: spacing.md },
  text: { flex: 1 },
  title: { fontSize: fontSizes.md, fontWeight: "600" },
  subtitle: { fontSize: fontSizes.sm, marginTop: 2 }
});
