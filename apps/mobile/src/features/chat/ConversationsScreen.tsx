import React from "react";
import { FlatList, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useQuery } from "@tanstack/react-query";
import { chatApi } from "../../api/endpoints";
import type { Conversation } from "../../api/types";
import { errorMessage, formatListTime } from "../../lib/format";
import { useAuth } from "../../providers/AuthProvider";
import { keys } from "../../providers/queryClient";
import { useRealtimeConnected } from "../../providers/RealtimeProvider";
import { useTheme } from "../../theme";
import { Avatar, CountBadge, EmptyState, ErrorState, Header, LoadingState, Screen, SectionTitle, Text } from "../../ui";

export function useConversations(filter: "new" | "active") {
  const live = useRealtimeConnected();
  return useQuery({
    queryKey: keys.conversations(filter),
    queryFn: () => chatApi.list(filter, 50),
    refetchInterval: live ? false : 15_000
  });
}

export function ConversationsScreen() {
  const navigation = useNavigation();
  const { user } = useAuth();
  const { colors, radii, spacing } = useTheme();
  const fresh = useConversations("new");
  const active = useConversations("active");

  if (fresh.isLoading || active.isLoading) {
    return (
      <Screen>
        <Header large title="Matchs" />
        <LoadingState />
      </Screen>
    );
  }
  if (active.isError || fresh.isError) {
    return (
      <Screen>
        <Header large title="Matchs" />
        <ErrorState
          message={errorMessage(active.error ?? fresh.error)}
          onRetry={() => {
            void fresh.refetch();
            void active.refetch();
          }}
        />
      </Screen>
    );
  }

  const open = (c: Conversation) => navigation.navigate("Chat", { conversationId: c.id });
  const newMatches = fresh.data ?? [];
  const conversations = active.data ?? [];

  return (
    <Screen>
      <Header large title="Matchs" />
      <FlatList
        contentContainerStyle={{ paddingBottom: spacing.xl, flexGrow: 1 }}
        data={conversations}
        keyExtractor={(c) => c.id}
        ListEmptyComponent={
          newMatches.length === 0 ? (
            <EmptyState
              icon="chatbubbles-outline"
              message="Like des profils dans Découvrir : quand c’est réciproque, vous pouvez discuter ici."
              title="Pas encore de match"
            />
          ) : (
            <View style={{ padding: spacing.lg }}>
              <Text tone="muted">Envoie le premier message à l’un de tes nouveaux matchs !</Text>
            </View>
          )
        }
        ListHeaderComponent={
          newMatches.length ? (
            <View style={{ gap: spacing.sm, paddingVertical: spacing.sm }}>
              <View style={{ paddingHorizontal: spacing.lg }}>
                <SectionTitle>Nouveaux matchs</SectionTitle>
              </View>
              <ScrollView contentContainerStyle={{ paddingHorizontal: spacing.lg, gap: spacing.md }} horizontal showsHorizontalScrollIndicator={false}>
                {newMatches.map((c) => (
                  <Pressable
                    accessibilityLabel={`Nouveau match avec ${c.user.firstName}`}
                    key={c.id}
                    onPress={() => open(c)}
                    style={styles.newMatch}
                    testID="new-match"
                  >
                    <Avatar name={c.user.firstName} ring size={68} uri={c.user.photoUrl} />
                    <Text numberOfLines={1} variant="caption">
                      {c.user.firstName}
                    </Text>
                  </Pressable>
                ))}
              </ScrollView>
              {conversations.length ? (
                <View style={{ paddingHorizontal: spacing.lg, paddingTop: spacing.sm }}>
                  <SectionTitle>Messages</SectionTitle>
                </View>
              ) : null}
            </View>
          ) : null
        }
        onRefresh={() => {
          void fresh.refetch();
          void active.refetch();
        }}
        refreshing={fresh.isRefetching || active.isRefetching}
        renderItem={({ item }) => {
          const mine = item.lastMessage?.senderId === user?.id;
          return (
            <Pressable
              accessibilityLabel={`Conversation avec ${item.user.firstName}${item.unreadCount ? `, ${item.unreadCount} non lus` : ""}`}
              onPress={() => open(item)}
              style={({ pressed }) => [
                styles.row,
                { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm, backgroundColor: pressed ? colors.surfaceAlt : "transparent", gap: spacing.md }
              ]}
              testID="conversation-row"
            >
              <Avatar name={item.user.firstName} size={56} uri={item.user.photoUrl} />
              <View style={[styles.flex, { borderBottomColor: colors.border, borderBottomWidth: StyleSheet.hairlineWidth, paddingBottom: spacing.sm, gap: 2 }]}>
                <View style={styles.between}>
                  <Text variant="bodyStrong">{item.user.firstName}</Text>
                  {item.lastMessage ? (
                    <Text tone={item.unreadCount ? "primary" : "subtle"} variant="caption">
                      {formatListTime(item.lastMessage.createdAt)}
                    </Text>
                  ) : null}
                </View>
                <View style={styles.between}>
                  <Text numberOfLines={1} style={styles.flex} tone={item.unreadCount ? "default" : "muted"} variant={item.unreadCount ? "bodyStrong" : "body"}>
                    {mine ? "Toi : " : ""}
                    {item.lastMessage?.body}
                  </Text>
                  <View style={{ marginLeft: spacing.xs, borderRadius: radii.pill }}>
                    <CountBadge count={item.unreadCount} />
                  </View>
                </View>
              </View>
            </Pressable>
          );
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  row: { flexDirection: "row", alignItems: "center" },
  between: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  newMatch: { width: 72, alignItems: "center", gap: 6 }
});
