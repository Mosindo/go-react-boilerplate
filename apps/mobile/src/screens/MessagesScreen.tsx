import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { chatApi } from "../api/endpoints";
import type { Conversation } from "../api/types";
import { errorMessage } from "../lib/errors";
import { formatTimestamp } from "../lib/dates";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParams, TabParams } from "../navigation/types";
import { useTheme } from "../theme";
import { EmptyState, ErrorState, LoadingState, PhotoImage, Screen, Text } from "../ui";
import { useAuth } from "../auth/AuthContext";

export function useConversations() {
  return useQuery({
    queryKey: queryKeys.conversations,
    queryFn: chatApi.conversations,
    staleTime: 15_000
  });
}

export function MessagesScreen({ navigation }: BottomTabScreenProps<TabParams, "Messages">) {
  const t = useTheme();
  const { user } = useAuth();
  const q = useConversations();
  const stack = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const open = (c: Conversation) =>
    stack.navigate("Chat", {
      conversationId: c.id,
      peerId: c.peer.id,
      peerName: c.peer.firstName,
      peerPhotoId: c.peer.mainPhotoId
    });

  if (q.isLoading)
    return (
      <Screen edges={["top"]}>
        <LoadingState />
      </Screen>
    );
  if (q.isError)
    return (
      <Screen edges={["top"]}>
        <ErrorState message={errorMessage(q.error)} onRetry={() => void q.refetch()} />
      </Screen>
    );

  const all = q.data ?? [];
  const fresh = all.filter((c) => !c.lastMessage);
  const talking = all.filter((c) => c.lastMessage);

  return (
    <Screen edges={["top"]} padded={false}>
      <View style={styles.header}>
        <Text variant="title">Messages</Text>
      </View>
      {all.length === 0 ? (
        <EmptyState
          icon="chatbubbles-outline"
          title="Pas encore de match"
          message="Quand l'intérêt est réciproque, votre conversation apparaît ici."
          actionLabel="Découvrir des profils"
          onAction={() => navigation.navigate("Discover")}
        />
      ) : (
        <FlatList
          data={talking}
          keyExtractor={(c) => c.id}
          refreshControl={
            <RefreshControl
              refreshing={q.isRefetching}
              onRefresh={() => void q.refetch()}
              tintColor={t.primary}
            />
          }
          ListHeaderComponent={
            fresh.length > 0 ? (
              <View style={styles.strip}>
                <Text variant="label" tone="muted" style={styles.stripTitle}>
                  Nouveaux matchs
                </Text>
                <FlatList
                  horizontal
                  showsHorizontalScrollIndicator={false}
                  data={fresh}
                  keyExtractor={(c) => c.id}
                  contentContainerStyle={styles.stripList}
                  renderItem={({ item }) => (
                    <Pressable
                      accessibilityRole="button"
                      accessibilityLabel={`Nouveau match avec ${item.peer.firstName}`}
                      onPress={() => open(item)}
                      style={styles.fresh}
                    >
                      <PhotoImage
                        photoId={item.peer.mainPhotoId}
                        style={[styles.freshPhoto, { borderColor: t.primary }]}
                      />
                      <Text variant="caption" numberOfLines={1}>
                        {item.peer.firstName}
                      </Text>
                    </Pressable>
                  )}
                />
              </View>
            ) : null
          }
          ListEmptyComponent={
            fresh.length > 0 ? (
              <Text tone="muted" style={styles.hint}>
                Lancez la conversation avec l'un de vos nouveaux matchs 👋
              </Text>
            ) : null
          }
          renderItem={({ item }) => {
            const last = item.lastMessage;
            const mine = last?.senderId === user?.id;
            return (
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={`Conversation avec ${item.peer.firstName}${item.unreadCount ? `, ${item.unreadCount} non lus` : ""}`}
                onPress={() => open(item)}
                style={[styles.row, { borderColor: t.border }]}
              >
                <PhotoImage photoId={item.peer.mainPhotoId} style={styles.avatar} />
                <View style={styles.rowBody}>
                  <View style={styles.rowTop}>
                    <Text variant="heading" numberOfLines={1} style={styles.flex}>
                      {item.peer.firstName}
                    </Text>
                    <Text variant="caption" tone="muted">
                      {last ? formatTimestamp(last.createdAt) : ""}
                    </Text>
                  </View>
                  <View style={styles.rowTop}>
                    <Text
                      tone={item.unreadCount ? "default" : "muted"}
                      numberOfLines={1}
                      style={[styles.flex, item.unreadCount ? styles.bold : null]}
                    >
                      {mine ? "Vous : " : ""}
                      {last?.body}
                    </Text>
                    {item.unreadCount > 0 ? (
                      <View style={[styles.badge, { backgroundColor: t.primary }]}>
                        <Text variant="caption" tone="onPrimary" style={styles.badgeText}>
                          {item.unreadCount > 99 ? "99+" : item.unreadCount}
                        </Text>
                      </View>
                    ) : null}
                  </View>
                </View>
              </Pressable>
            );
          }}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { paddingHorizontal: 20, paddingVertical: 8 },
  strip: { paddingBottom: 8 },
  stripTitle: { paddingHorizontal: 20, paddingBottom: 8 },
  stripList: { paddingHorizontal: 16, gap: 14 },
  fresh: { width: 72, alignItems: "center", gap: 4 },
  freshPhoto: { width: 68, height: 68, borderRadius: 34, borderWidth: 2 },
  hint: { padding: 20, textAlign: "center" },
  row: {
    flexDirection: "row",
    gap: 14,
    paddingHorizontal: 20,
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  avatar: { width: 56, height: 56, borderRadius: 28 },
  rowBody: { flex: 1, gap: 2, justifyContent: "center" },
  rowTop: { flexDirection: "row", alignItems: "center", gap: 8 },
  flex: { flex: 1 },
  bold: { fontWeight: "700" },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: 11,
    paddingHorizontal: 6,
    alignItems: "center",
    justifyContent: "center"
  },
  badgeText: { fontWeight: "700", fontSize: 12 }
});
