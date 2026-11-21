import React from "react";
import { Platform, Pressable, Switch, View } from "react-native";
import { makeStyles, useTheme } from "../theme";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type ListRowProps = {
  icon?: IconName;
  label: string;
  description?: string;
  onPress?: () => void;
  tone?: "default" | "danger";
  value?: boolean;
  onValueChange?: (value: boolean) => void;
  testID?: string;
  divider?: boolean;
};

export function ListRow({
  icon,
  label,
  description,
  onPress,
  tone = "default",
  value,
  onValueChange,
  testID,
  divider = true
}: ListRowProps) {
  const styles = useStyles();
  const theme = useTheme();
  const isSwitch = typeof value === "boolean" && onValueChange;
  const content = (
    <>
      {icon ? <Icon name={icon} size={20} color={tone === "danger" ? "danger" : "textMuted"} /> : null}
      <View style={styles.texts}>
        <Text variant="bodyStrong" tone={tone === "danger" ? "danger" : "default"}>
          {label}
        </Text>
        {description ? (
          <Text variant="caption" tone="muted">
            {description}
          </Text>
        ) : null}
      </View>
      {isSwitch ? (
        <Switch
          testID={testID}
          accessibilityLabel={label}
          value={value}
          onValueChange={onValueChange}
          trackColor={{ true: theme.colors.primary, false: theme.colors.borderStrong }}
          thumbColor={theme.colors.surface}
          {...(Platform.OS === "web" ? ({ activeThumbColor: theme.colors.surface } as object) : {})}
        />
      ) : onPress ? (
        <Icon name="chevron-forward" size={18} color="textSubtle" />
      ) : null}
    </>
  );
  if (isSwitch || !onPress) {
    return <View style={[styles.row, divider ? styles.divider : null]}>{content}</View>;
  }
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [styles.row, divider ? styles.divider : null, pressed ? styles.pressed : null]}
    >
      {content}
    </Pressable>
  );
}

const useStyles = makeStyles((t) => ({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: t.spacing.md,
    paddingHorizontal: t.spacing.lg,
    paddingVertical: t.spacing.md,
    minHeight: 56
  },
  divider: { borderBottomWidth: 1, borderBottomColor: t.colors.border },
  pressed: { backgroundColor: t.colors.surfaceAlt },
  texts: { flex: 1, gap: 2 }
}));
