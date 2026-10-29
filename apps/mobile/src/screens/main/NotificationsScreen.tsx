import { Ionicons } from "@expo/vector-icons";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import {
  fetchNotifications,
  markAllNotificationsRead,
  markNotificationRead
} from "../../api/notifications";
import type { NotificationItem } from "../../api/types";
import { Button } from "../../components/Button";
import { EmptyView, ErrorView, LoadingView } from "../../components/StateViews";
import { queryKeys } from "../../hooks/queries";
import { errorMessage } from "../../hooks/useAsyncAction";
import { formatRelative } from "../../lib/format";
import type { MainTabParamList, RootStackParamList } from "../../navigation/types";
import { fontSizes, radii, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

type Props = CompositeScreenProps<
  BottomTabScreenProps<MainTabParamList, "Notifications">,
  NativeStackScreenProps<RootStackParamList>
>;

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => fetchNotifications(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextBefore ?? undefined,
    refetchOnMount: "always"
  });
}

function describe(n: NotificationItem): { icon: "heart" | "chatbubble"; text: string } {
  const name = n.data.firstName ?? "Quelqu'un";
  if (n.type === "match") {
    return { icon: "heart", text: `Nouveau match avec ${name} !` };
  }
  return { icon: "chatbubble", text: `${name} : ${n.data.preview ?? "vous a écrit"}` };
}

export default function NotificationsScreen({ navigation }: Props) {
  const theme = useTheme();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const items = useMemo(
    () => query.data?.pages.flatMap((p) => p.notifications) ?? [],
    [query.data]
  );
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const refresh = () => queryClient.invalidateQueries({ queryKey: queryKeys.notifications });

  const open = async (n: NotificationItem) => {
    if (!n.readAt) {
      void markNotificationRead(n.id)
        .then(refresh)
        .catch(() => undefined);
    }
    const conversationId = n.data.conversationId;
    if (conversationId) {
      navigation.navigate("Chat", {
        conversationId,
        user: { id: n.data.userId ?? "", firstName: n.data.firstName ?? "", photoUrl: null }
      });
    }
  };

  return (
    <SafeAreaView edges={["top"]} style={[styles.flex, { backgroundColor: theme.background }]}>
      <View style={styles.header}>
        <Text accessibilityRole="header" style={[styles.title, { color: theme.text }]}>
          Notifications
        </Text>
        {unread > 0 ? (
          <Button
            onPress={() => void markAllNotificationsRead().then(refresh)}
            style={styles.markAll}
            title="Tout lire"
            variant="ghost"
          />
        ) : null}
      </View>
      {query.isLoading ? (
        <LoadingView />
      ) : query.isError ? (
        <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : items.length === 0 ? (
        <EmptyView
          icon="notifications-outline"
          message="Vos nouveaux matchs et messages apparaîtront ici."
          title="Rien de neuf pour l'instant"
        />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          refreshControl={
            <RefreshControl
              onRefresh={() => void query.refetch()}
              refreshing={query.isRefetching && !query.isFetchingNextPage}
              tintColor={theme.primary}
            />
          }
          renderItem={({ item }) => {
            const { icon, text } = describe(item);
            return (
              <Pressable
                accessibilityRole="button"
                onPress={() => void open(item)}
                style={({ pressed }) => [
                  styles.row,
                  {
                    backgroundColor: pressed
                      ? theme.surfaceMuted
                      : item.readAt
                        ? "transparent"
                        : theme.primarySoft
                  }
                ]}
              >
                <View style={[styles.icon, { backgroundColor: theme.primarySoft }]}>
                  <Ionicons color={theme.primary} name={icon} size={20} />
                </View>
                <View style={styles.text}>
                  <Text
                    numberOfLines={2}
                    style={[
                      styles.message,
                      { color: theme.text, fontWeight: item.readAt ? "400" : "700" }
                    ]}
                  >
                    {text}
                  </Text>
                  <Text style={[styles.time, { color: theme.textMuted }]}>
                    {formatRelative(item.createdAt)}
                  </Text>
                </View>
              </Pressable>
            );
          }}
        />
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingRight: spacing.sm
  },
  title: {
    fontSize: 30,
    fontWeight: "700",
    letterSpacing: -0.5,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  },
  markAll: { minHeight: 40, paddingHorizontal: spacing.md },
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  },
  icon: {
    width: 40,
    height: 40,
    borderRadius: radii.pill,
    alignItems: "center",
    justifyContent: "center",
    marginRight: spacing.md
  },
  text: { flex: 1 },
  message: { fontSize: fontSizes.md, lineHeight: 22 },
  time: { fontSize: fontSizes.xs, marginTop: 2 }
});
