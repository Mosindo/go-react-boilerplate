import { Ionicons } from "@expo/vector-icons";
import { StyleSheet, View } from "react-native";
import { radii, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export function Notice({ message, kind = "info" }: { message: string; kind?: "info" | "error" | "success" }) {
  const { colors } = useTheme();
  const palette = {
    info: { bg: colors.surfaceMuted, fg: colors.text, icon: "information-circle-outline" as const },
    error: { bg: colors.dangerSoft, fg: colors.danger, icon: "alert-circle-outline" as const },
    success: { bg: colors.primarySoft, fg: colors.success, icon: "checkmark-circle-outline" as const }
  }[kind];
  return (
    <View accessibilityRole="alert" style={[styles.box, { backgroundColor: palette.bg }]}>
      <Ionicons name={palette.icon} size={20} color={palette.fg} />
      <Text variant="caption" style={{ color: palette.fg, flex: 1 }}>
        {message}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  box: { flexDirection: "row", gap: spacing.sm, alignItems: "flex-start", padding: spacing.md, borderRadius: radii.md }
});
