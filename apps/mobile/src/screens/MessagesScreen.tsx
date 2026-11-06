import React from "react";
import { FlatList, Pressable, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useInfiniteQuery } from "@tanstack/react-query";
import { chatApi } from "../api/services";
import type { Conversation } from "../api/types";
import { Avatar } from "../components/Avatar";
import { Badge, Divider, EmptyState, ErrorState, Loader, Screen, Text } from "../components/ui";
import { fr } from "../i18n/fr";
import { formatListTime } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme";
import { useAuth } from "../auth/AuthContext";

export function useConversations() {
  return useInfiniteQuery({
    queryKey: ["conversations"],
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextBefore,
    refetchOnWindowFocus: true
  });
}

export function MessagesScreen() {
  const { colors } = useTheme();
  const { user } = useAuth();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const query = useConversations();
  const conversations = query.data?.pages.flatMap((p) => p.conversations) ?? [];
  const fresh = conversations.filter((c) => !c.lastMessage);
  const talking = conversations.filter((c) => c.lastMessage);

  const open = (c: Conversation) =>
    navigation.navigate("Chat", { conversationId: c.id, name: c.peer.firstName });

  const header = (
    <View>
      {fresh.length > 0 ? (
        <View style={{ paddingTop: spacing.md, gap: spacing.sm }}>
          <Text variant="label" muted style={{ paddingHorizontal: spacing.lg }}>
            {fr.messages.newMatches}
          </Text>
          <FlatList
            horizontal
            showsHorizontalScrollIndicator={false}
            data={fresh}
            keyExtractor={(c) => c.id}
            contentContainerStyle={{ paddingHorizontal: spacing.lg, gap: spacing.lg }}
            renderItem={({ item }) => (
              <Pressable
                testID={`new-match-${item.peer.firstName}`}
                accessibilityRole="button"
                accessibilityLabel={`${item.peer.firstName} — ${fr.messages.sayHi}`}
                onPress={() => open(item)}
                style={{ alignItems: "center", gap: 6 }}
              >
                <Avatar name={item.peer.firstName} photoId={item.peer.mainPhotoId} size={68} />
                <Text variant="caption">{item.peer.firstName}</Text>
              </Pressable>
            )}
          />
        </View>
      ) : null}
      {talking.length > 0 ? (
        <Text
          variant="label"
          muted
          style={{
            paddingHorizontal: spacing.lg,
            paddingTop: spacing.lg,
            paddingBottom: spacing.sm
          }}
        >
          {fr.messages.conversations}
        </Text>
      ) : null}
    </View>
  );

  return (
    <Screen>
      <View style={{ paddingHorizontal: spacing.lg, paddingTop: spacing.sm }}>
        <Text variant="title">{fr.messages.title}</Text>
      </View>
      {query.isLoading ? <Loader /> : null}
      {query.isError ? <ErrorState onRetry={() => void query.refetch()} /> : null}
      {query.data && conversations.length === 0 ? (
        <EmptyState glyph="💬" title={fr.messages.emptyTitle} body={fr.messages.emptyBody} />
      ) : null}
      {query.data && conversations.length > 0 ? (
        <FlatList
          testID="conversation-list"
          data={talking}
          keyExtractor={(c) => c.id}
          ListHeaderComponent={header}
          onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
          onEndReachedThreshold={0.4}
          refreshing={query.isRefetching}
          onRefresh={() => void query.refetch()}
          ItemSeparatorComponent={Divider}
          renderItem={({ item }) => {
            const last = item.lastMessage;
            const mine = last?.senderId === user?.id;
            return (
              <Pressable
                testID={`conversation-${item.peer.firstName}`}
                accessibilityRole="button"
                accessibilityLabel={`${item.peer.firstName}${item.unreadCount ? `, ${item.unreadCount} non lus` : ""}`}
                onPress={() => open(item)}
                style={({ pressed }) => ({
                  flexDirection: "row",
                  alignItems: "center",
                  gap: spacing.md,
                  paddingHorizontal: spacing.lg,
                  paddingVertical: spacing.md,
                  backgroundColor: pressed ? colors.surfaceAlt : "transparent"
                })}
              >
                <Avatar name={item.peer.firstName} photoId={item.peer.mainPhotoId} />
                <View style={{ flex: 1, gap: 2 }}>
                  <View style={{ flexDirection: "row", justifyContent: "space-between" }}>
                    <Text variant="heading" numberOfLines={1} style={{ flex: 1 }}>
                      {item.peer.firstName}
                    </Text>
                    <Text variant="caption" muted>
                      {formatListTime(item.activityAt)}
                    </Text>
                  </View>
                  <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.sm }}>
                    <Text
                      numberOfLines={1}
                      muted={item.unreadCount === 0}
                      style={{ flex: 1, fontWeight: item.unreadCount > 0 ? "700" : "400" }}
                    >
                      {last
                        ? `${mine ? fr.messages.youPrefix : ""}${last.body}`
                        : fr.messages.sayHi}
                    </Text>
                    <Badge count={item.unreadCount} />
                  </View>
                </View>
              </Pressable>
            );
          }}
        />
      ) : null}
    </Screen>
  );
}
