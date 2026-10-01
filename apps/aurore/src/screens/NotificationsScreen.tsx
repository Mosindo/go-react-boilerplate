import React from "react";
import { FlatList, Pressable, RefreshControl, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { chatApi, discoverApi, notificationsApi } from "../api/endpoints";
import { keys } from "../api/keys";
import type { AppNotification } from "../api/types";
import { Avatar } from "../components/AuthImage";
import { useFeedback } from "../components/Feedback";
import { errorMessage } from "../components/forms";
import { Button, EmptyState, ErrorState, Loading, Screen, Separator, Text } from "../components/ui";
import { formatMessageTime } from "../lib/dates";
import type { RootStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme/theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

function describe(n: AppNotification): string {
  const who = n.actorName || "Quelqu'un";
  return n.type === "match" ? `Nouveau match avec ${who} !` : `${who} vous a écrit`;
}

export function NotificationsScreen() {
  const t = useTheme();
  const nav = useNavigation<Nav>();
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const query = useQuery({ queryKey: keys.notifications, queryFn: notificationsApi.list });

  const refresh = () => {
    void qc.invalidateQueries({ queryKey: keys.notifications });
    void qc.invalidateQueries({ queryKey: keys.unread });
  };
  const readAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: refresh, onError: (e) => toast(errorMessage(e), "error") });
  const readOne = useMutation({ mutationFn: notificationsApi.markRead, onSuccess: refresh });

  const open = async (n: AppNotification) => {
    if (!n.isRead) readOne.mutate(n.id);
    try {
      if (n.type === "message") {
        const convs = await qc.fetchQuery({ queryKey: keys.conversations, queryFn: chatApi.conversations, staleTime: 5000 });
        const c = convs.find((x) => x.id === n.refId);
        if (c) return nav.navigate("Chat", { conversationId: c.id, userId: c.user.userId, name: c.user.firstName });
      } else {
        const ms = await qc.fetchQuery({ queryKey: keys.matches, queryFn: discoverApi.matches, staleTime: 5000 });
        const m = ms.find((x) => x.id === n.refId);
        if (m) return nav.navigate("Chat", { conversationId: m.conversationId, userId: m.user.userId, name: m.user.firstName });
      }
      toast("Cette conversation n'existe plus.");
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  if (query.isLoading) return <Screen><Loading /></Screen>;
  if (query.error) return <Screen><ErrorState message={(query.error as Error).message} onRetry={() => void query.refetch()} /></Screen>;
  const items = query.data?.notifications ?? [];

  return (
    <Screen padded={false}>
      <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between", paddingHorizontal: spacing.lg, paddingTop: spacing.sm, paddingBottom: spacing.md }}>
        <Text variant="title">Notifications</Text>
        {items.some((n) => !n.isRead) ? <Button label="Tout lire" variant="ghost" onPress={() => readAll.mutate()} testID="notifications-read-all" style={{ minHeight: 40 }} /> : null}
      </View>
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} />}
        ItemSeparatorComponent={Separator}
        ListEmptyComponent={<EmptyState icon="notifications-outline" title="Rien de nouveau" message="Vos nouveaux matchs et messages apparaîtront ici." />}
        contentContainerStyle={{ flexGrow: 1 }}
        renderItem={({ item }) => (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={describe(item)}
            testID="notification-row"
            onPress={() => void open(item)}
            style={({ pressed }) => ({ flexDirection: "row", alignItems: "center", gap: spacing.md, padding: spacing.lg, backgroundColor: !item.isRead ? t.primarySoft : pressed ? t.surfaceAlt : "transparent" })}
          >
            <Avatar path={item.actorThumbUrl} size={48} />
            <View style={{ flex: 1 }}>
              <Text style={{ fontWeight: item.isRead ? "400" : "700" }}>{describe(item)}</Text>
              <Text variant="caption" muted>{formatMessageTime(item.createdAt)}</Text>
            </View>
          </Pressable>
        )}
      />
    </Screen>
  );
}
