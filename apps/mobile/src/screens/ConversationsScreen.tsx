import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { ConversationSummary } from "../api/types";
import { ConversationRow } from "../components/ConversationRow";
import { PhotoImage } from "../components/PhotoImage";
import { Screen } from "../components/Screen";
import { useConversations } from "../hooks/queries";
import { useCurrentUser } from "../hooks/useAuth";
import type { MainStackParamList } from "../navigation/types";
import { EmptyView, ErrorView } from "../shared/feedback";
import { Loader, Text, colors, spacing } from "../shared/ui";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export default function ConversationsScreen() {
  const navigation = useNavigation<Nav>();
  const user = useCurrentUser();
  const query = useConversations();

  const all = useMemo(() => query.data?.pages.flatMap((p) => p.conversations) ?? [], [query.data]);
  const fresh = all.filter((c) => c.lastMessage === null);
  const talking = all.filter((c) => c.lastMessage !== null);

  const open = (c: ConversationSummary) =>
    navigation.navigate("Conversation", { conversationId: c.id, matchId: c.matchId, userId: c.user.userId, name: c.user.firstName });

  return (
    <Screen edges={["top", "left", "right"]} contentStyle={styles.content} testID="messages-screen">
      <Text style={styles.title} variant="title" weight="bold">
        Messages
      </Text>

      {query.isLoading ? <Loader fullScreen label="Chargement…" /> : null}
      {query.isError && !query.data ? (
        <ErrorView actionLabel="Réessayer" message="Impossible de charger vos conversations." onAction={() => void query.refetch()} title="Oups" />
      ) : null}

      {query.data ? (
        <FlatList
          ListEmptyComponent={
            fresh.length === 0 ? (
              <EmptyView
                actionLabel="Découvrir des profils"
                message="Quand vous et quelqu'un vous aimerez mutuellement, la conversation apparaîtra ici."
                onAction={() => navigation.navigate("Tabs")}
                testID="messages-empty"
                title="Pas encore de match"
              />
            ) : null
          }
          ListHeaderComponent={
            fresh.length > 0 ? (
              <View style={styles.newMatches}>
                <Text tone="secondary" variant="eyebrow" weight="bold" style={styles.eyebrow}>
                  Nouveaux matchs
                </Text>
                <FlatList
                  data={fresh}
                  horizontal
                  keyExtractor={(c) => c.id}
                  renderItem={({ item }) => (
                    <Pressable accessibilityLabel={`Nouveau match : ${item.user.firstName}`} accessibilityRole="button" onPress={() => open(item)} style={styles.match} testID={`new-match-${item.user.firstName}`}>
                      <PhotoImage fallbackLabel={item.user.firstName} path={item.user.thumbUrl} style={styles.matchPhoto} />
                      <Text numberOfLines={1} variant="label" weight="semibold">
                        {item.user.firstName}
                      </Text>
                    </Pressable>
                  )}
                  showsHorizontalScrollIndicator={false}
                />
              </View>
            ) : null
          }
          contentContainerStyle={styles.list}
          data={talking}
          keyExtractor={(c) => c.id}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          onEndReachedThreshold={0.4}
          refreshControl={<RefreshControl onRefresh={() => void query.refetch()} refreshing={query.isRefetching && !query.isFetchingNextPage} tintColor={colors.primary} />}
          renderItem={({ item }) => <ConversationRow conversation={item} currentUserId={user.id} onPress={() => open(item)} />}
        />
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { padding: 0, paddingTop: spacing.lg },
  title: { paddingHorizontal: spacing.lg },
  list: { paddingBottom: 100, flexGrow: 1 },
  newMatches: { paddingVertical: spacing.md, gap: spacing.sm },
  eyebrow: { paddingHorizontal: spacing.lg },
  match: { width: 84, alignItems: "center", gap: spacing.xs, marginLeft: spacing.lg },
  matchPhoto: { width: 68, height: 68, borderRadius: 34, borderWidth: 2, borderColor: colors.primary }
});
