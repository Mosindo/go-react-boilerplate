import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { notificationApi } from "../api/endpoints";
import type { AppNotification } from "../api/types";
import { keys } from "../features/keys";
import { formatListDate } from "../lib/dates";
import { describeError } from "../lib/validation";
import { useNav } from "../nav/Navigation";
import { useTheme } from "../theme/theme";
import { Button, EmptyState, ErrorState, LoadingView, PhotoImage, Screen, Text, useToast } from "../ui";

function label(n: AppNotification): string {
  return n.type === "match" ? `Vous avez un match avec ${n.actorName} !` : `${n.actorName} vous a envoyé un message`;
}

export default function NotificationsScreen() {
  const { colors } = useTheme();
  const nav = useNav();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: keys.notifications, queryFn: () => notificationApi.list() });

  const refresh = () => void queryClient.invalidateQueries({ queryKey: keys.notifications });
  const readOne = useMutation({ mutationFn: notificationApi.markRead, onSuccess: refresh });
  const readAll = useMutation({
    mutationFn: notificationApi.markAllRead,
    onSuccess: refresh,
    onError: (e) => toast.show(describeError(e), "error")
  });

  const open = (n: AppNotification) => {
    if (!n.readAt) readOne.mutate(n.id);
    if (n.conversationId) {
      nav.push({
        name: "chat",
        conversationId: n.conversationId,
        matchId: n.matchId,
        peer: { userId: n.actorId, firstName: n.actorName, photoUrl: n.actorPhotoUrl }
      });
    }
  };

  const items = query.data?.notifications ?? [];
  return (
    <Screen
      title="Notifications"
      onBack={nav.back}
      right={items.some((n) => !n.readAt) ? <Button title="Tout lire" variant="ghost" onPress={() => readAll.mutate()} /> : undefined}
    >
      {query.isPending ? <LoadingView /> : null}
      {query.isError ? <ErrorState message={describeError(query.error)} onRetry={() => void query.refetch()} /> : null}
      {query.isSuccess && items.length === 0 ? (
        <EmptyState title="Rien de neuf" message="Vos matchs et nouveaux messages apparaîtront ici." />
      ) : null}
      {query.isSuccess && items.length > 0 ? (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              onPress={() => open(item)}
              style={[styles.row, { borderBottomColor: colors.border, backgroundColor: item.readAt ? "transparent" : colors.surfaceAlt }]}
            >
              <PhotoImage url={item.actorPhotoUrl} name={item.actorName} circle style={styles.avatar} />
              <View style={styles.body}>
                <Text variant={item.readAt ? "body" : "label"}>{label(item)}</Text>
                <Text variant="caption" muted>
                  {formatListDate(item.createdAt, new Date())}
                </Text>
              </View>
            </Pressable>
          )}
        />
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: 12, padding: 14, borderBottomWidth: StyleSheet.hairlineWidth },
  avatar: { width: 48, height: 48 },
  body: { flex: 1, gap: 2 }
});
