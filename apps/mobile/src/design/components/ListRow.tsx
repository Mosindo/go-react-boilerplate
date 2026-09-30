import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../ThemeProvider";
import { Text } from "./Text";

type Props = {
  icon?: keyof typeof Ionicons.glyphMap;
  title: string;
  subtitle?: string;
  onPress?: () => void;
  right?: React.ReactNode;
  destructive?: boolean;
  testID?: string;
};

export function ListRow({ icon, title, subtitle, onPress, right, destructive, testID }: Props) {
  const { colors } = useTheme();
  const tint = destructive ? colors.danger : colors.text;
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
      testID={testID}
    >
      {icon ? <Ionicons color={destructive ? colors.danger : colors.textMuted} name={icon} size={22} /> : null}
      <View style={styles.text}>
        <Text style={{ color: tint }} variant="body">
          {title}
        </Text>
        {subtitle ? (
          <Text tone="muted" variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right ?? (onPress ? <Ionicons color={colors.textSubtle} name="chevron-forward" size={18} /> : null)}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: 14, paddingVertical: 14, paddingHorizontal: 16, minHeight: 56 },
  text: { flex: 1, gap: 2 }
});
