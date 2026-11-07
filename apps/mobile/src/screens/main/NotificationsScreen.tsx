import React, { useEffect } from "react";
import { FlatList, Pressable } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AppText, Avatar, EmptyState, ErrorState, Loading, Row, Screen } from "../../components/ui";
import { notificationsApi } from "../../api/services";
import { errorMessage } from "../../api/http";
import { formatTime } from "../../utils/format";
import { spacing, useTheme } from "../../theme/theme";
import type { MainStackParams } from "../../navigation/types";
import type { NotificationItem } from "../../api/types";

export default function NotificationsScreen() {
  const { colors } = useTheme();
  const nav = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["notifications"], queryFn: notificationsApi.list });

  // Opening the screen counts as reading; the unread markers stay visible for this visit.
  const hasUnread = (q.data?.unreadCount ?? 0) > 0;
  useEffect(() => {
    if (!hasUnread) return;
    const timer = setTimeout(() => {
      notificationsApi
        .readAll()
        .then(() => qc.invalidateQueries({ queryKey: ["notifications"], refetchType: "none" }))
        .catch(() => undefined);
    }, 1500);
    return () => clearTimeout(timer);
  }, [hasUnread, qc]);

  const open = (n: NotificationItem) => {
    if (!n.actorId) return;
    if (n.type === "match" && n.refId)
      nav.navigate("Chat", { conversationId: n.refId, peerName: n.actorName ?? "", peerId: n.actorId });
    else nav.navigate("ProfileDetail", { userId: n.actorId, canAct: true });
  };

  return (
    <Screen>
      {q.isLoading ? (
        <Loading />
      ) : q.isError ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => void q.refetch()} />
      ) : (q.data?.notifications ?? []).length === 0 ? (
        <EmptyState title="Rien de nouveau" message="Vos nouveaux likes et matchs apparaîtront ici." />
      ) : (
        <FlatList
          data={q.data?.notifications}
          keyExtractor={(n) => n.id}
          refreshing={q.isRefetching}
          onRefresh={() => void q.refetch()}
          renderItem={({ item }) => (
            <Pressable
              testID={`notification-${item.type}`}
              accessibilityRole="button"
              onPress={() => open(item)}
              style={{ paddingVertical: spacing.md, borderBottomWidth: 1, borderColor: colors.border }}
            >
              <Row>
                <Avatar name={item.actorName ?? "?"} uri={item.photoUrl} />
                <AppText style={{ flex: 1, fontWeight: item.readAt ? "400" : "700" }}>
                  {item.type === "match"
                    ? `Vous avez matché avec ${item.actorName ?? "quelqu'un"} !`
                    : `${item.actorName ?? "Quelqu'un"} vous a liké.`}
                </AppText>
                <AppText variant="muted">{formatTime(item.createdAt)}</AppText>
              </Row>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}
