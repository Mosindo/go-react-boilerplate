import React from "react";
import { FlatList, Pressable, RefreshControl, Text, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiErrorMessage } from "../../api/client";
import { notificationApi } from "../../api/endpoints";
import type { NotificationItem } from "../../api/types";
import { AppText, Avatar, Button, EmptyState, ErrorState, Loading, Screen } from "../../components/ui";
import { formatActivity } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme/theme";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export default function NotificationsScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const query = useInfiniteQuery({
    queryKey: ["notifications", "list"],
    queryFn: ({ pageParam }) => notificationApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => (typeof last.nextCursor === "string" ? last.nextCursor : undefined)
  });
  const items = query.data?.pages.flatMap((p) => p.items) ?? [];
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const refresh = () => void queryClient.invalidateQueries({ queryKey: ["notifications"] });
  const markAll = useMutation({ mutationFn: notificationApi.markAllRead, onSuccess: refresh });
  const markOne = useMutation({ mutationFn: notificationApi.markRead, onSuccess: refresh });

  const open = (n: NotificationItem) => {
    if (!n.read) markOne.mutate(n.id);
    if (n.data.matchId && n.data.userId) {
      navigation.navigate("Chat", { matchId: n.data.matchId, partnerId: n.data.userId, partnerName: n.data.firstName ?? "" });
    }
  };

  const label = (n: NotificationItem) =>
    n.type === "match"
      ? `Nouveau match avec ${n.data.firstName ?? "quelqu'un"} 🎉`
      : `${n.data.firstName ?? "Quelqu'un"} : ${n.data.preview ?? "nouveau message"}`;

  return (
    <Screen edges={["top"]} padded={false}>
      <View
        style={{
          flexDirection: "row",
          justifyContent: "space-between",
          alignItems: "center",
          paddingHorizontal: spacing.lg,
          paddingVertical: spacing.sm
        }}
      >
        <AppText variant="heading">Notifications</AppText>
        {unread > 0 ? <Button label="Tout marquer comme lu" variant="ghost" onPress={() => markAll.mutate()} /> : null}
      </View>
      {query.isLoading ? <Loading /> : null}
      {query.isError ? <ErrorState message={apiErrorMessage(query.error)} onRetry={() => void query.refetch()} /> : null}
      {!query.isLoading && !query.isError && items.length === 0 ? (
        <EmptyState emoji="🔔" title="Rien de neuf" message="Vos nouveaux matchs et messages apparaîtront ici." />
      ) : null}
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        refreshControl={
          <RefreshControl
            refreshing={query.isRefetching && !query.isFetchingNextPage}
            onRefresh={() => void query.refetch()}
            tintColor={colors.primary}
          />
        }
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
        renderItem={({ item }) => (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={label(item)}
            onPress={() => open(item)}
            style={{
              flexDirection: "row",
              alignItems: "center",
              gap: spacing.md,
              padding: spacing.lg,
              backgroundColor: item.read ? "transparent" : colors.primarySoft
            }}
          >
            <Avatar url={item.data.photoUrl ?? null} name={item.data.firstName ?? "?"} size={48} />
            <View style={{ flex: 1 }}>
              <Text numberOfLines={2} style={{ color: colors.text, fontWeight: item.read ? "400" : "700" }}>
                {label(item)}
              </Text>
              <Text style={{ color: colors.textMuted, fontSize: 12 }}>{formatActivity(item.createdAt)}</Text>
            </View>
          </Pressable>
        )}
      />
    </Screen>
  );
}
