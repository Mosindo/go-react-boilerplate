import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { FlatList } from "react-native";
import { markAllNotificationsRead, markNotificationRead } from "../api/platform";
import type { AppNotification } from "../api/types";
import { useMatches, useNotifications } from "../hooks/queries";
import type { RootStackParamList } from "../navigation/types";
import { queryKeys } from "../realtime/cache";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Button, NotificationItem } from "../shared/ui";

export default function NotificationsScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const { data, isLoading, isError, error, refetch } = useNotifications();
  const matches = useMatches();

  const refresh = () => queryClient.invalidateQueries({ queryKey: queryKeys.notifications });

  const open = async (n: AppNotification) => {
    if (!n.isRead) {
      await markNotificationRead(n.id).catch(() => undefined);
      void refresh();
    }
    const conversationId = typeof n.data.conversationId === "string" ? n.data.conversationId : null;
    const match = matches.data?.find((m) => m.conversationId === conversationId);
    if (match) {
      navigation.navigate("Chat", { conversationId: match.conversationId, matchId: match.id, userId: match.user.id, name: match.user.firstName, photoPath: match.user.photos[0]?.url });
    } else if (typeof n.data.userId === "string") {
      navigation.navigate("ProfileDetail", { userId: n.data.userId });
    }
  };

  return (
    <ScreenContainer testID="notifications-screen">
      <Header
        title="Notifications"
        onBack={() => navigation.goBack()}
        right={
          data && data.unreadCount > 0 ? (
            <Button
              testID="read-all"
              label="Tout lire"
              variant="ghost"
              compact
              onPress={async () => {
                await markAllNotificationsRead();
                void refresh();
              }}
            />
          ) : undefined
        }
      />
      {isLoading ? <LoadingView /> : null}
      {isError ? <ErrorView error={error} onRetry={() => void refetch()} /> : null}
      {data && data.notifications.length === 0 ? <EmptyView icon="notifications-outline" title="Rien de neuf" message="Vos nouveaux matchs et messages apparaîtront ici." /> : null}
      {data && data.notifications.length > 0 ? (
        <FlatList data={data.notifications} keyExtractor={(n) => n.id} renderItem={({ item }) => <NotificationItem notification={item} onPress={() => void open(item)} />} />
      ) : null}
    </ScreenContainer>
  );
}
