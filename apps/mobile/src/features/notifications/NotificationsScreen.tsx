import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { chatApi, notificationsApi } from "../../api/endpoints";
import type { AppNotification, Conversation, NotificationsPage } from "../../api/types";
import type { MainStackParamList } from "../../navigation/types";
import { keys } from "../../state/queryClient";
import { useNotifications } from "../../state/queries";
import { spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, EmptyState, ErrorState, Icon, LoadingState, Screen, Text, useToast } from "../../ui";
import { relativeTime } from "../../utils/format";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export function NotificationsScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const qc = useQueryClient();
  const toast = useToast();
  const { data, isLoading, isError, error, refetch, isRefetching } = useNotifications();

  const markLocal = (id?: string) =>
    qc.setQueryData<NotificationsPage>(keys.notifications, (p) =>
      p
        ? {
            unreadCount: id ? Math.max(0, p.unreadCount - (p.notifications.find((n) => n.id === id && !n.isRead) ? 1 : 0)) : 0,
            notifications: p.notifications.map((n) => (!id || n.id === id ? { ...n, isRead: true } : n))
          }
        : p
    );

  const readAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: () => markLocal() });

  const open = async (n: AppNotification) => {
    if (!n.isRead) {
      markLocal(n.id);
      notificationsApi.markRead(n.id).catch(() => undefined);
    }
    const matchId = n.data?.matchId;
    if (!matchId) return;
    let conv = qc.getQueryData<Conversation[]>(keys.conversations)?.find((c) => c.matchId === matchId);
    if (!conv) {
      const list = await qc.fetchQuery({ queryKey: keys.conversations, queryFn: () => chatApi.conversations() });
      conv = list.find((c) => c.matchId === matchId);
    }
    if (conv) {
      navigation.navigate("Conversation", { matchId, user: conv.user });
    } else {
      toast({ title: "Cette conversation n'est plus disponible", tone: "error" });
    }
  };

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message={errorMessage(error)} onRetry={refetch} />;

  return (
    <Screen padded={false}>
      <View style={styles.header}>
        <Text accessibilityRole="header" variant="title">
          Activité
        </Text>
        {data?.unreadCount ? <Button label="Tout marquer comme lu" loading={readAll.isPending} onPress={() => readAll.mutate()} size="sm" variant="ghost" /> : null}
      </View>
      <FlatList
        ListEmptyComponent={<EmptyState body="Vos nouveaux matchs et messages apparaîtront ici." icon="notifications-outline" title="Rien de neuf" />}
        contentContainerStyle={styles.list}
        data={data?.notifications ?? []}
        keyExtractor={(n) => n.id}
        refreshControl={<RefreshControl onRefresh={refetch} refreshing={isRefetching} tintColor={colors.primary} />}
        renderItem={({ item }) => (
          <Pressable
            accessibilityRole="button"
            accessibilityState={{ selected: !item.isRead }}
            onPress={() => open(item)}
            style={({ pressed }) => [styles.row, pressed ? { backgroundColor: colors.surfaceMuted } : null]}
          >
            <View style={[styles.icon, { backgroundColor: item.type === "match" ? colors.primarySoft : colors.accentSoft }]}>
              <Icon color={item.type === "match" ? colors.primary : colors.accent} name={item.type === "match" ? "heart" : "chatbubble"} size={20} />
            </View>
            <View style={styles.text}>
              <Text weight={item.isRead ? "500" : "700"}>{item.title}</Text>
              <Text numberOfLines={2} tone="muted" variant="caption">
                {item.body}
              </Text>
              <Text tone="subtle" variant="caption">
                {relativeTime(item.createdAt)}
              </Text>
            </View>
            {!item.isRead ? <View accessibilityLabel="Non lu" style={[styles.dot, { backgroundColor: colors.primary }]} /> : null}
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", paddingHorizontal: spacing.lg, paddingTop: spacing.md, paddingBottom: spacing.xs },
  list: { flexGrow: 1, paddingBottom: spacing.xxl },
  row: { flexDirection: "row", gap: spacing.md, alignItems: "center", paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  icon: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" },
  text: { flex: 1, gap: 2 },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
