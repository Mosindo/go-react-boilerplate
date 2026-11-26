import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import type { Conversation } from "../../api/types";
import { useConversations, useMatches } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import type { AppStackParamList, TabParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { Avatar, Badge, EmptyState, ErrorState, Header, LoadingState, Screen, Text } from "../../ui";
import { formatRelative } from "../../utils/format";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Messages">, NativeStackScreenProps<AppStackParamList>>;

export function MessagesScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { user } = useAuth();
  const conversations = useConversations();
  const matches = useMatches();

  const newMatches = useMemo(() => (matches.data?.pages.flatMap((p) => p.matches) ?? []).filter((m) => !m.hasMessages), [matches.data]);
  const items = useMemo(() => conversations.data?.pages.flatMap((p) => p.conversations) ?? [], [conversations.data]);

  const refresh = () => {
    void conversations.refetch();
    void matches.refetch();
  };

  const renderConversation = ({ item }: { item: Conversation }) => {
    const unread = item.unreadCount > 0;
    const mine = item.lastMessage?.senderId === user?.id;
    return (
      <Pressable
        accessibilityLabel={`Conversation avec ${item.profile.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
        accessibilityRole="button"
        onPress={() => navigation.navigate("Chat", { conversationId: item.id })}
        style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
        testID={`conversation-${item.id}`}
      >
        <Avatar name={item.profile.firstName} size={56} url={item.profile.photos[0]?.url} />
        <View style={styles.rowText}>
          <View style={styles.rowTop}>
            <Text numberOfLines={1} style={styles.flex} variant="bodyStrong">
              {item.profile.firstName}
            </Text>
            {item.lastMessage ? (
              <Text tone={unread ? "primary" : "subtle"} variant="caption">
                {formatRelative(item.lastMessage.createdAt)}
              </Text>
            ) : null}
          </View>
          <View style={styles.rowTop}>
            <Text numberOfLines={1} style={styles.flex} tone={unread ? "default" : "muted"} variant={unread ? "label" : "caption"}>
              {mine ? "Vous : " : ""}
              {item.lastMessage?.body}
            </Text>
            <Badge count={item.unreadCount} />
          </View>
        </View>
      </Pressable>
    );
  };

  const header = (
    <View>
      {newMatches.length > 0 ? (
        <View style={styles.matchesBlock}>
          <Text tone="subtle" variant="overline">
            Nouveaux matchs
          </Text>
          <ScrollView contentContainerStyle={styles.matchesRow} horizontal showsHorizontalScrollIndicator={false}>
            {newMatches.map((m) => (
              <Pressable
                accessibilityLabel={`Écrire à ${m.profile.firstName}`}
                accessibilityRole="button"
                key={m.id}
                onPress={() => navigation.navigate("Chat", { conversationId: m.conversationId })}
                style={styles.matchItem}
                testID={`new-match-${m.profile.userId}`}
              >
                <Avatar name={m.profile.firstName} ring size={68} url={m.profile.photos[0]?.url} />
                <Text numberOfLines={1} variant="label">
                  {m.profile.firstName}
                </Text>
              </Pressable>
            ))}
          </ScrollView>
        </View>
      ) : null}
      {items.length > 0 ? (
        <Text style={styles.sectionTitle} tone="subtle" variant="overline">
          Conversations
        </Text>
      ) : null}
    </View>
  );

  let body: React.ReactNode;
  if (conversations.isLoading || matches.isLoading) {
    body = <LoadingState />;
  } else if (conversations.isError) {
    body = <ErrorState message={errorMessage(conversations.error)} onRetry={refresh} />;
  } else {
    body = (
      <FlatList
        contentContainerStyle={styles.list}
        data={items}
        keyExtractor={(c) => c.id}
        ListEmptyComponent={
          newMatches.length > 0 ? (
            <Text center style={styles.hint} tone="muted">
              Choisissez un match ci-dessus pour envoyer le premier message.
            </Text>
          ) : (
            <EmptyState
              actionLabel="Découvrir des profils"
              icon="chatbubbles-outline"
              message="Quand l'intérêt est réciproque, vous pourrez discuter ici."
              onAction={() => navigation.navigate("Tabs", { screen: "Discover" })}
              title="Aucune conversation"
            />
          )
        }
        ListHeaderComponent={header}
        onEndReached={() => {
          if (conversations.hasNextPage && !conversations.isFetchingNextPage) void conversations.fetchNextPage();
        }}
        refreshControl={<RefreshControl onRefresh={refresh} refreshing={conversations.isRefetching} tintColor={colors.primary} />}
        renderItem={renderConversation}
      />
    );
  }

  return (
    <Screen padded={false} testID="messages-screen">
      <View style={styles.padded}>
        <Header large title="Messages" />
      </View>
      {body}
    </Screen>
  );
}

const styles = StyleSheet.create({
  padded: { paddingHorizontal: spacing.lg },
  list: { flexGrow: 1, paddingBottom: spacing.xxl },
  matchesBlock: { gap: spacing.sm, paddingHorizontal: spacing.lg, paddingBottom: spacing.md },
  matchesRow: { gap: spacing.md },
  matchItem: { alignItems: "center", gap: 6, width: 76 },
  sectionTitle: { paddingHorizontal: spacing.lg, paddingBottom: spacing.xs },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  rowText: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  flex: { flex: 1 },
  hint: { padding: spacing.xl }
});
