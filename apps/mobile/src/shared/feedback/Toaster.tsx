import { Ionicons } from "@expo/vector-icons";
import { Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, spacing, useTheme } from "../../theme";
import { Text } from "../ui/Text";
import { dismissToast, useToasts } from "./store";

/** Renders transient feedback above the whole app (screen readers announce each message politely). */
export function Toaster() {
  const toasts = useToasts();
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  if (toasts.length === 0) return null;
  return (
    <View pointerEvents="box-none" style={[styles.host, { top: insets.top + spacing.sm }]}>
      {toasts.map((toast) => {
        const palette = {
          success: { bg: colors.success, icon: "checkmark-circle" as const },
          error: { bg: colors.danger, icon: "alert-circle" as const },
          info: { bg: colors.accent, icon: "information-circle" as const }
        }[toast.kind];
        const fg = toast.kind === "info" ? colors.background : "#FFFFFF";
        return (
          <Pressable
            key={toast.id}
            accessibilityRole="alert"
            accessibilityLiveRegion="polite"
            onPress={() => dismissToast(toast.id)}
            style={[styles.toast, { backgroundColor: palette.bg }]}
          >
            <Ionicons name={palette.icon} size={20} color={fg} />
            <Text variant="caption" style={{ color: fg, flex: 1 }}>
              {toast.message}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  host: { position: "absolute", left: spacing.lg, right: spacing.lg, gap: spacing.sm, alignItems: "center", zIndex: 100 },
  toast: { flexDirection: "row", alignItems: "center", gap: spacing.sm, padding: spacing.md, borderRadius: radii.md, width: "100%", maxWidth: 520 }
});
