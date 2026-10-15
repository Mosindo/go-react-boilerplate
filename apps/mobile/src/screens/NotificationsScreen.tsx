import React from "react";
import { FlatList, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markAllNotificationsRead, markNotificationRead } from "../api/platform";
import type { AppNotification } from "../api/types";
import { queryKeys, useNotifications } from "../hooks/queries";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, NotificationItem, Text, spacing } from "../shared/ui";
import { useTheme } from "../theme";

export default function NotificationsScreen() {
  const palette = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const refresh = () => queryClient.invalidateQueries({ queryKey: queryKeys.notifications });

  const readOne = useMutation({ mutationFn: markNotificationRead, onSettled: refresh });
  const readAll = useMutation({ mutationFn: markAllNotificationsRead, onSettled: refresh });

  const open = (notification: AppNotification) => {
    if (!notification.isRead) {
      readOne.mutate(notification.id);
    }
    const { conversationId, userId, firstName } = notification.data;
    if (conversationId && userId) {
      navigation.navigate("Thread", { conversationId, peerId: userId, peerName: firstName ?? "Conversation" });
    }
  };

  const items = query.data?.notifications ?? [];
  const unread = query.data?.unreadCount ?? 0;

  return (
    <ScreenContainer testID="notifications-screen">
      <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between", minHeight: 56 }}>
        <Text accessibilityRole="header" variant="heading" weight="bold">
          Notifications
        </Text>
        {unread > 0 ? (
          <Button label="Tout lire" loading={readAll.isPending} onPress={() => readAll.mutate()} size="sm" variant="ghost" />
        ) : null}
      </View>

      {query.isLoading ? (
        <LoadingView fullScreen />
      ) : query.isError ? (
        <ErrorView error={query.error} onRetry={() => void query.refetch()} />
      ) : items.length === 0 ? (
        <EmptyView
          icon={<Ionicons color={palette.textSubtle} name="notifications-outline" size={56} />}
          message="Vos nouveaux matchs et messages apparaîtront ici."
          style={{ flex: 1 }}
          testID="notifications-empty"
          title="Rien de nouveau"
        />
      ) : (
        <FlatList
          contentContainerStyle={{ gap: spacing.sm, paddingBottom: spacing.xl }}
          data={items}
          keyExtractor={(n) => n.id}
          onRefresh={() => void query.refetch()}
          refreshing={query.isRefetching}
          renderItem={({ item }) => <NotificationItem notification={item} onPress={() => open(item)} />}
          showsVerticalScrollIndicator={false}
        />
      )}
    </ScreenContainer>
  );
}
