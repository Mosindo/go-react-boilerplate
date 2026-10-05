import { Ionicons } from "@expo/vector-icons";
import { StyleSheet, View } from "react-native";
import { spacing, useTheme } from "../../theme";
import { Button } from "../ui/Button";
import { Text } from "../ui/Text";

/** Friendly empty state: what is missing, why, and what to do about it. */
export function EmptyView({ icon = "sparkles-outline", title, message, actionLabel, onAction }: { icon?: keyof typeof Ionicons.glyphMap; title: string; message?: string; actionLabel?: string; onAction?: () => void }) {
  const { colors } = useTheme();
  return (
    <View style={styles.box}>
      <View style={[styles.icon, { backgroundColor: colors.primarySoft }]}>
        <Ionicons name={icon} size={32} color={colors.primary} />
      </View>
      <Text variant="heading" center>
        {title}
      </Text>
      {message ? (
        <Text tone="muted" center>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} compact /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  box: { alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xxl, flex: 1 },
  icon: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center" }
});
