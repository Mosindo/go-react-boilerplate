import { Ionicons } from "@expo/vector-icons";
import { StyleSheet, View } from "react-native";
import type { AppNotification } from "../../api/types";
import { formatRelative } from "../../lib/dates";
import { useTheme } from "../../theme";
import { ListItem } from "./ListItem";
import { Text } from "./Text";

export function NotificationItem({ notification, onPress }: { notification: AppNotification; onPress: () => void }) {
  const { colors } = useTheme();
  const icon = notification.type === "match" ? "heart" : "chatbubble-ellipses";
  return (
    <ListItem
      onPress={onPress}
      title={notification.title}
      subtitle={notification.body}
      subtitleStrong={!notification.isRead}
      accessibilityLabel={`${notification.isRead ? "" : "Non lue. "}${notification.title}. ${notification.body}`}
      leading={
        <View style={[styles.icon, { backgroundColor: colors.primarySoft }]}>
          <Ionicons name={icon} size={20} color={colors.primary} />
        </View>
      }
      trailing={
        <>
          <Text variant="caption" tone="muted">
            {formatRelative(notification.createdAt)}
          </Text>
          {!notification.isRead ? <View style={[styles.dot, { backgroundColor: colors.primary }]} /> : null}
        </>
      }
    />
  );
}

const styles = StyleSheet.create({
  icon: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
