import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import type { ConversationPreview, MatchPreview } from "../api/types";
import { useConversations, useMatches } from "../hooks/queries";
import { useCurrentUser } from "../hooks/useAuth";
import { formatRelative } from "../lib/dates";
import type { RootStackParamList } from "../navigation/types";
import { queryKeys } from "../realtime/cache";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Avatar, Badge, ListItem, Text } from "../shared/ui";
import { layout, spacing } from "../theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

/** "Messages" tab: new matches (no message yet) on top, then conversations. */
export default function MessagesScreen() {
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const me = useCurrentUser();
  const matches = useMatches();
  const conversations = useConversations();
  const [refreshing, setRefreshing] = useState(false);

  const refresh = async () => {
    setRefreshing(true);
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queryKeys.matches }),
      queryClient.invalidateQueries({ queryKey: queryKeys.conversations })
    ]);
    setRefreshing(false);
  };

  const openMatch = (m: MatchPreview) =>
    navigation.navigate("Chat", { conversationId: m.conversationId, matchId: m.id, userId: m.user.id, name: m.user.firstName, photoPath: m.user.photos[0]?.url });

  const openConversation = (c: ConversationPreview) =>
    navigation.navigate("Chat", { conversationId: c.id, matchId: c.matchId, userId: c.user.id, name: c.user.firstName, photoPath: c.user.photos[0]?.url });

  if (matches.isLoading || conversations.isLoading) {
    return (
      <SafeAreaLayout edges={["top"]}>
        <LoadingView />
      </SafeAreaLayout>
    );
  }
  if (matches.isError || conversations.isError) {
    return (
      <SafeAreaLayout edges={["top"]}>
        <ErrorView error={matches.error ?? conversations.error} onRetry={() => void refresh()} />
      </SafeAreaLayout>
    );
  }

  const newMatches = (matches.data ?? []).filter((m) => !m.lastMessage);
  const list = conversations.data ?? [];

  return (
    <SafeAreaLayout edges={["top"]}>
      <View style={styles.page}>
        <FlatList
          testID="conversation-list"
          data={list}
          keyExtractor={(c) => c.id}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void refresh()} />}
          ListHeaderComponent={
            <View style={styles.header}>
              <Text variant="title" style={styles.title}>
                Messages
              </Text>
              {newMatches.length > 0 ? (
                <View>
                  <Text variant="label" tone="muted" style={styles.section}>
                    NOUVEAUX MATCHS
                  </Text>
                  <FlatList
                    horizontal
                    showsHorizontalScrollIndicator={false}
                    data={newMatches}
                    keyExtractor={(m) => m.id}
                    contentContainerStyle={styles.rail}
                    renderItem={({ item }) => (
                      <Pressable testID={`new-match-${item.user.firstName}`} style={styles.railItem} accessibilityRole="button" accessibilityLabel={`Nouveau match : ${item.user.firstName}. Écrire`} onPress={() => openMatch(item)}>
                        <Avatar name={item.user.firstName} path={item.user.photos[0]?.url} size={68} />
                        <Text variant="caption" numberOfLines={1} style={styles.railName}>
                          {item.user.firstName}
                        </Text>
                      </Pressable>
                    )}
                  />
                </View>
              ) : null}
              {list.length > 0 ? (
                <Text variant="label" tone="muted" style={styles.section}>
                  CONVERSATIONS
                </Text>
              ) : null}
            </View>
          }
          ListEmptyComponent={
            newMatches.length === 0 ? (
              <EmptyView icon="chatbubbles-outline" title="Pas encore de conversation" message="Likez des profils : dès que l'intérêt est réciproque, vous pourrez discuter ici." />
            ) : (
              <Text tone="muted" center style={styles.hint}>
                Choisissez un match ci-dessus pour envoyer le premier message.
              </Text>
            )
          }
          renderItem={({ item }) => {
            const last = item.lastMessage;
            const mine = last?.senderUserId === me.id;
            return (
              <ListItem
                testID={`conversation-${item.user.firstName}`}
                leading={<Avatar name={item.user.firstName} path={item.user.photos[0]?.url} />}
                title={item.user.firstName}
                subtitle={last ? `${mine ? "Vous : " : ""}${last.content}` : undefined}
                subtitleStrong={item.unreadCount > 0}
                trailing={
                  <>
                    <Text variant="caption" tone="muted">
                      {last ? formatRelative(last.createdAt) : ""}
                    </Text>
                    <Badge count={item.unreadCount} />
                  </>
                }
                onPress={() => openConversation(item)}
              />
            );
          }}
        />
      </View>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  page: { flex: 1, width: "100%", maxWidth: layout.maxContentWidth, alignSelf: "center" },
  header: { gap: spacing.md, paddingTop: spacing.sm },
  title: { paddingHorizontal: spacing.lg },
  section: { paddingHorizontal: spacing.lg, marginBottom: spacing.xs },
  rail: { paddingHorizontal: spacing.lg, gap: spacing.lg },
  railItem: { alignItems: "center", width: 72, gap: spacing.xs },
  railName: { maxWidth: 72 },
  hint: { padding: spacing.xl }
});
