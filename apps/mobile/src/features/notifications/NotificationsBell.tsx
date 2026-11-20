import React from "react";
import { View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useNotifications } from "./queries";
import { CountBadge, IconButton } from "../../ui";

export function NotificationsBell() {
  const navigation = useNavigation();
  const notifications = useNotifications();
  const unread = notifications.data?.unreadCount ?? 0;
  return (
    <View>
      <IconButton
        icon="notifications-outline"
        label={`Notifications${unread ? `, ${unread} non lues` : ""}`}
        onPress={() => navigation.navigate("Notifications")}
        testID="notifications-bell"
      />
      <View pointerEvents="none" style={{ position: "absolute", top: 2, right: 2 }}>
        <CountBadge count={unread} />
      </View>
    </View>
  );
}
