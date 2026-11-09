import React, { useState } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { chatApi, safetyApi } from "../api/endpoints";
import type { Conversation } from "../api/types";
import { ReportSheet } from "../features/ReportSheet";
import { keys } from "../features/keys";
import { formatListDate } from "../lib/dates";
import { describeError } from "../lib/validation";
import { useNav } from "../nav/Navigation";
import { useTheme } from "../theme/theme";
import { ActionSheet, EmptyState, ErrorState, LoadingView, PhotoImage, Screen, Text, useToast } from "../ui";

export default function MessagesScreen() {
  const { colors } = useTheme();
  const nav = useNav();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [target, setTarget] = useState<Conversation | null>(null);
  const [reporting, setReporting] = useState<Conversation | null>(null);

  const query = useInfiniteQuery({
    queryKey: keys.conversationList,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => (last.length >= 30 ? last[last.length - 1].sortAt : undefined)
  });

  const refreshAll = () => {
    void queryClient.invalidateQueries({ queryKey: keys.conversations });
    void queryClient.invalidateQueries({ queryKey: keys.discover });
  };

  const hide = useMutation({
    mutationFn: (c: Conversation) => chatApi.hide(c.id),
    onSuccess: refreshAll,
    onError: (e) => toast.show(describeError(e), "error")
  });
  const block = useMutation({
    mutationFn: (c: Conversation) => safetyApi.block(c.peer.userId),
    onSuccess: () => {
      toast.show("Personne bloquée.");
      refreshAll();
    },
    onError: (e) => toast.show(describeError(e), "error")
  });

  const all = query.data?.pages.flat() ?? [];
  const fresh = all.filter((c) => !c.lastMessage);
  const talking = all.filter((c) => c.lastMessage);

  const open = (c: Conversation) => nav.push({ name: "chat", conversationId: c.id, peer: c.peer, matchId: c.matchId });

  let body: React.ReactNode;
  if (query.isPending) body = <LoadingView />;
  else if (query.isError) body = <ErrorState message={describeError(query.error)} onRetry={() => void query.refetch()} />;
  else if (all.length === 0)
    body = (
      <EmptyState
        title="Pas encore de match"
        message="Likez des profils : dès que l'intérêt est réciproque, la conversation apparaît ici."
      />
    );
  else
    body = (
      <FlatList
        data={talking}
        keyExtractor={(c) => c.id}
        refreshControl={
          <RefreshControl
            refreshing={query.isRefetching && !query.isFetchingNextPage}
            onRefresh={() => void query.refetch()}
            tintColor={colors.primary}
          />
        }
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
        ListHeaderComponent={
          fresh.length > 0 ? (
            <View style={styles.freshWrap}>
              <Text variant="label" muted style={styles.sectionTitle}>
                Nouveaux matchs
              </Text>
              <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.freshRow}>
                {fresh.map((c) => (
                  <Pressable
                    key={c.id}
                    accessibilityRole="button"
                    accessibilityLabel={`Nouveau match : ${c.peer.firstName}`}
                    onPress={() => open(c)}
                    onLongPress={() => setTarget(c)}
                    style={styles.freshItem}
                  >
                    <PhotoImage
                      url={c.peer.photoUrl}
                      name={c.peer.firstName}
                      circle
                      style={[styles.freshAvatar, { borderColor: colors.primary }]}
                    />
                    <Text variant="caption" numberOfLines={1}>
                      {c.peer.firstName}
                    </Text>
                  </Pressable>
                ))}
              </ScrollView>
              {talking.length > 0 ? (
                <Text variant="label" muted style={styles.sectionTitle}>
                  Messages
                </Text>
              ) : null}
            </View>
          ) : null
        }
        ListEmptyComponent={
          fresh.length > 0 ? (
            <Text muted style={styles.hint}>
              Appuyez sur un match pour écrire le premier message.
            </Text>
          ) : null
        }
        renderItem={({ item }) => <ConversationRow conversation={item} onPress={() => open(item)} onLongPress={() => setTarget(item)} />}
      />
    );

  return (
    <Screen title="Messages" noBottomInset>
      {body}
      <ActionSheet
        visible={!!target}
        title={target?.peer.firstName}
        actions={
          target
            ? [
                { label: "Supprimer la conversation", onPress: () => hide.mutate(target) },
                { label: "Signaler", destructive: true, onPress: () => setReporting(target) },
                { label: "Bloquer", destructive: true, onPress: () => block.mutate(target) }
              ]
            : []
        }
        onClose={() => setTarget(null)}
      />
      {reporting ? (
        <ReportSheet
          visible
          userId={reporting.peer.userId}
          name={reporting.peer.firstName}
          onClose={() => setReporting(null)}
          onDone={() => {
            setReporting(null);
            refreshAll();
          }}
        />
      ) : null}
    </Screen>
  );
}

function ConversationRow({
  conversation,
  onPress,
  onLongPress
}: {
  conversation: Conversation;
  onPress: () => void;
  onLongPress: () => void;
}) {
  const { colors } = useTheme();
  const last = conversation.lastMessage;
  const unread = conversation.unreadCount > 0;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${conversation.peer.firstName}${unread ? `, ${conversation.unreadCount} non lus` : ""}`}
      onPress={onPress}
      onLongPress={onLongPress}
      style={[styles.row, { borderBottomColor: colors.border }]}
    >
      <PhotoImage url={conversation.peer.photoUrl} name={conversation.peer.firstName} circle style={styles.avatar} />
      <View style={styles.rowBody}>
        <View style={styles.rowTop}>
          <Text variant="label" style={styles.flex} numberOfLines={1}>
            {conversation.peer.firstName}
          </Text>
          {last ? (
            <Text variant="caption" muted>
              {formatListDate(last.createdAt, new Date())}
            </Text>
          ) : null}
        </View>
        <Text variant="caption" muted={!unread} numberOfLines={1} style={unread ? styles.bold : undefined}>
          {last?.body ?? ""}
        </Text>
      </View>
      {unread ? (
        <View style={[styles.badge, { backgroundColor: colors.primary }]}>
          <Text variant="caption" color={colors.onPrimary}>
            {conversation.unreadCount}
          </Text>
        </View>
      ) : null}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  avatar: { width: 56, height: 56 },
  rowBody: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: 8 },
  bold: { fontWeight: "700" },
  badge: { minWidth: 22, height: 22, borderRadius: 11, alignItems: "center", justifyContent: "center", paddingHorizontal: 6 },
  freshWrap: { paddingTop: 8 },
  sectionTitle: { paddingHorizontal: 16, paddingVertical: 8 },
  freshRow: { paddingHorizontal: 12, gap: 12 },
  freshItem: { alignItems: "center", gap: 4, width: 76 },
  freshAvatar: { width: 68, height: 68, borderWidth: 2 },
  hint: { padding: 24, textAlign: "center" }
});
