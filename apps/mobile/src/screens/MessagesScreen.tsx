import { useState } from "react";
import { FlatList, Pressable, RefreshControl, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { Badge, Avatar, EmptyState, ErrorState, Loading, Screen, Text } from "../components/ui";
import { errorMessage } from "../lib/errors";
import { formatRelative } from "../lib/format";
import { useAuth } from "../state/auth";
import { useConversations, useMatches } from "../state/queries";
import { radius, spacing, useColors } from "../theme";

export function MessagesScreen() {
  const navigation = useNavigation();
  const colors = useColors();
  const { user } = useAuth();
  const conversations = useConversations();
  const matches = useMatches();
  const [refreshing, setRefreshing] = useState(false);

  const refresh = async () => {
    setRefreshing(true);
    await Promise.all([conversations.refetch(), matches.refetch()]);
    setRefreshing(false);
  };

  if (conversations.isLoading) {
    return (
      <Screen edges={["top"]}>
        <Loading />
      </Screen>
    );
  }
  if (conversations.isError) {
    return (
      <Screen edges={["top"]}>
        <ErrorState message={errorMessage(conversations.error)} onRetry={() => void conversations.refetch()} />
      </Screen>
    );
  }

  const items = conversations.data?.pages.flatMap((p) => p.items) ?? [];
  const fresh = (matches.data ?? []).filter((m) => !m.hasMessages);

  const header = (
    <View>
      <Text style={{ paddingHorizontal: spacing.lg, paddingTop: spacing.sm }} variant="title">
        Messages
      </Text>
      {fresh.length > 0 ? (
        <View style={{ marginTop: spacing.lg }}>
          <Text muted style={{ paddingHorizontal: spacing.lg, marginBottom: spacing.sm }} variant="label">
            Nouveaux matchs
          </Text>
          <FlatList
            contentContainerStyle={{ paddingHorizontal: spacing.lg }}
            data={fresh}
            horizontal
            keyExtractor={(m) => m.id}
            renderItem={({ item }) => (
              <Pressable
                accessibilityLabel={`Nouveau match avec ${item.firstName}`}
                accessibilityRole="button"
                onPress={() =>
                  navigation.navigate("Chat", {
                    conversationId: item.conversationId,
                    userId: item.userId,
                    name: item.firstName,
                  })
                }
                style={{ alignItems: "center", marginRight: spacing.lg }}
              >
                <View style={{ borderWidth: 2, borderColor: colors.primary, borderRadius: 40, padding: 2 }}>
                  <Avatar name={item.firstName} photoId={item.photoId} size={64} />
                </View>
                <Text style={{ marginTop: spacing.xs }} variant="caption">
                  {item.firstName}
                </Text>
              </Pressable>
            )}
            showsHorizontalScrollIndicator={false}
          />
        </View>
      ) : null}
      {items.length > 0 ? (
        <Text
          muted
          style={{ paddingHorizontal: spacing.lg, marginTop: spacing.lg, marginBottom: spacing.sm }}
          variant="label"
        >
          Conversations
        </Text>
      ) : null}
    </View>
  );

  return (
    <Screen edges={["top"]} padded={false}>
      <FlatList
        ListEmptyComponent={
          <EmptyState
            icon="chatbubbles-outline"
            message={
              fresh.length > 0
                ? "Lancez la conversation avec l'un de vos nouveaux matchs."
                : "Quand vous aurez un match, vous pourrez discuter ici."
            }
            title="Aucune conversation"
          />
        }
        ListHeaderComponent={header}
        contentContainerStyle={{ flexGrow: 1 }}
        data={items}
        keyExtractor={(c) => c.id}
        onEndReached={() => {
          if (conversations.hasNextPage && !conversations.isFetchingNextPage) {
            void conversations.fetchNextPage();
          }
        }}
        refreshControl={
          <RefreshControl onRefresh={() => void refresh()} refreshing={refreshing} tintColor={colors.primary} />
        }
        renderItem={({ item }) => {
          const last = item.lastMessage;
          const mine = last?.senderId === user?.id;
          return (
            <Pressable
              accessibilityRole="button"
              onPress={() =>
                navigation.navigate("Chat", { conversationId: item.id, userId: item.userId, name: item.firstName })
              }
              style={({ pressed }) => ({
                flexDirection: "row",
                alignItems: "center",
                paddingHorizontal: spacing.lg,
                paddingVertical: spacing.md,
                backgroundColor: pressed ? colors.surfaceAlt : "transparent",
              })}
            >
              <Avatar name={item.firstName} photoId={item.photoId} size={56} />
              <View style={{ flex: 1, marginLeft: spacing.md }}>
                <View style={{ flexDirection: "row", justifyContent: "space-between" }}>
                  <Text variant="label">{item.firstName}</Text>
                  {last ? (
                    <Text muted variant="caption">
                      {formatRelative(last.createdAt)}
                    </Text>
                  ) : null}
                </View>
                <View style={{ flexDirection: "row", alignItems: "center", marginTop: 2 }}>
                  <Text
                    muted={item.unreadCount === 0}
                    numberOfLines={1}
                    style={{ flex: 1, fontWeight: item.unreadCount > 0 ? "700" : "400" }}
                    variant="caption"
                  >
                    {last ? `${mine ? "Vous : " : ""}${last.body}` : ""}
                  </Text>
                  <Badge count={item.unreadCount} />
                </View>
              </View>
            </Pressable>
          );
        }}
        style={{ borderRadius: radius.md }}
      />
    </Screen>
  );
}
