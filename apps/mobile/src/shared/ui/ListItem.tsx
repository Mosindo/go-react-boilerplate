import type { ReactNode } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { spacing, useTheme } from "../../theme";
import { Text } from "./Text";

/** Row with a leading element, two text lines and trailing content (time, badge). */
export function ListItem({
  leading,
  title,
  subtitle,
  subtitleStrong,
  trailing,
  onPress,
  testID,
  accessibilityLabel
}: {
  leading?: ReactNode;
  title: string;
  subtitle?: string;
  subtitleStrong?: boolean;
  trailing?: ReactNode;
  onPress?: () => void;
  testID?: string;
  accessibilityLabel?: string;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      accessibilityLabel={accessibilityLabel ?? (subtitle ? `${title}. ${subtitle}` : title)}
      testID={testID}
      onPress={onPress}
      style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
    >
      {leading}
      <View style={styles.texts}>
        <Text variant="bodyStrong" numberOfLines={1}>
          {title}
        </Text>
        {subtitle ? (
          <Text variant="caption" tone={subtitleStrong ? "default" : "muted"} numberOfLines={1} style={subtitleStrong ? styles.strong : undefined}>
            {subtitle}
          </Text>
        ) : null}
      </View>
      {trailing ? <View style={styles.trailing}>{trailing}</View> : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.md, minHeight: 72 },
  texts: { flex: 1, gap: 2 },
  strong: { fontWeight: "700" },
  trailing: { alignItems: "flex-end", gap: spacing.xs }
});
