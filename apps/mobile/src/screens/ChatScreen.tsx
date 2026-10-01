import React from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { ConversationSummary, MatchSummary } from "../api/types";
import { useConversations, useMatches } from "../hooks/useData";
import { useAuth } from "../hooks/useAuth";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Avatar, CountBadge, ListItem, Text, spacing } from "../shared/ui";
import { formatRelativeTime } from "../utils/format";

type Nav = NativeStackNavigationProp<RootStackParamList>;

/** "Messages" tab: new matches on top, conversations below. */
export default function ChatScreen() {
  const navigation = useNavigation<Nav>();
  const { user } = useAuth();
  const conversations = useConversations();
  const matches = useMatches();

  const fresh: MatchSummary[] = (matches.data ?? []).filter((m) => !m.hasMessages);
  const items: ConversationSummary[] = conversations.data?.conversations ?? [];

  const open = (conversationId: string, profile: ConversationSummary["user"]) =>
    navigation.navigate("Conversation", { conversationId, user: profile });

  const refreshing = (conversations.isRefetching && !conversations.isLoading) || (matches.isRefetching && !matches.isLoading);
  const refresh = () => {
    void conversations.refetch();
    void matches.refetch();
  };

  if (conversations.isLoading) {
    return (
      <SafeAreaLayout edges={["top"]}>
        <LoadingView label="Chargement des messages…" />
      </SafeAreaLayout>
    );
  }
  if (conversations.isError) {
    return (
      <SafeAreaLayout edges={["top"]}>
        <ErrorView message="Impossible de charger vos conversations." onAction={refresh} />
      </SafeAreaLayout>
    );
  }

  return (
    <SafeAreaLayout edges={["top"]}>
      <FlatList
        testID="messages-screen"
        ListEmptyComponent={
          <EmptyView
            actionLabel="Découvrir des profils"
            icon="chatbubbles-outline"
            message={
              fresh.length > 0
                ? "Vous avez de nouveaux matchs : lancez la conversation en touchant l'un d'eux."
                : "Likez des profils : quand l'intérêt est réciproque, vous pourrez discuter ici."
            }
            onAction={() => navigation.navigate("Tabs", { screen: "Discover" })}
            title="Aucune conversation"
          />
        }
        ListHeaderComponent={
          <View style={styles.header}>
            <Text accessibilityRole="header" variant="title">
              Messages
            </Text>
            {fresh.length > 0 ? (
              <View style={styles.strip}>
                <Text variant="label" weight="bold">
                  Nouveaux matchs
                </Text>
                <ScrollView contentContainerStyle={styles.stripRow} horizontal showsHorizontalScrollIndicator={false}>
                  {fresh.map((m) => (
                    <Pressable
                      accessibilityLabel={`Nouveau match : ${m.user.firstName}`}
                      accessibilityRole="button"
                      key={m.id}
                      onPress={() => open(m.conversationId, m.user)}
                      style={styles.stripItem}
                    >
                      <Avatar name={m.user.firstName} path={m.user.photos[0]?.url} size="lg" />
                      <Text numberOfLines={1} variant="caption" weight="semibold">
                        {m.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
              </View>
            ) : null}
          </View>
        }
        contentContainerStyle={items.length === 0 ? styles.emptyContent : undefined}
        data={items}
        keyExtractor={(item) => item.id}
        refreshControl={<RefreshControl onRefresh={refresh} refreshing={refreshing} />}
        renderItem={({ item }) => {
          const mine = item.lastMessage.senderId === user?.id;
          return (
            <ListItem
              emphasized={item.unreadCount > 0}
              left={<Avatar name={item.user.firstName} path={item.user.photos[0]?.url} />}
              onPress={() => open(item.id, item.user)}
              right={
                <View style={styles.meta}>
                  <Text tone="subtle" variant="caption">
                    {formatRelativeTime(item.lastMessage.createdAt)}
                  </Text>
                  <CountBadge count={item.unreadCount} />
                </View>
              }
              subtitle={`${mine ? "Vous : " : ""}${item.lastMessage.body}`}
              title={item.user.firstName}
            />
          );
        }}
      />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  header: { paddingHorizontal: spacing.lg, paddingTop: spacing.sm, gap: spacing.md, paddingBottom: spacing.sm },
  strip: { gap: spacing.sm },
  stripRow: { gap: spacing.md },
  stripItem: { alignItems: "center", gap: spacing.xxs, width: 68 },
  meta: { alignItems: "flex-end", gap: spacing.xxs },
  emptyContent: { flexGrow: 1 }
});
