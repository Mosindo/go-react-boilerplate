import { StyleSheet, View } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";

/** Round counter used for unread messages and notifications. Renders nothing for 0. */
export function Badge({ count }: { count: number }) {
  const { colors } = useTheme();
  if (count <= 0) return null;
  return (
    <View
      accessibilityLabel={`${count} non lu${count > 1 ? "s" : ""}`}
      style={[styles.badge, { backgroundColor: colors.primary }]}
    >
      <Text variant="label" tone="onPrimary" style={styles.text}>
        {count > 99 ? "99+" : count}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: { minWidth: 22, height: 22, borderRadius: 11, paddingHorizontal: 6, alignItems: "center", justifyContent: "center" },
  text: { fontSize: 12, lineHeight: 16 }
});
