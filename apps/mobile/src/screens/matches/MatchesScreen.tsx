import React from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, Text, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useInfiniteQuery } from "@tanstack/react-query";
import { apiErrorMessage } from "../../api/client";
import { chatApi } from "../../api/endpoints";
import type { Conversation } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";
import { AppText, Avatar, Badge, EmptyState, ErrorState, Loading, Screen } from "../../components/ui";
import { formatActivity } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme/theme";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export default function MatchesScreen() {
  const { colors } = useTheme();
  const { user } = useAuth();
  const navigation = useNavigation<Nav>();
  const query = useInfiniteQuery({
    queryKey: ["conversations", "list"],
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => (typeof last.nextCursor === "string" ? last.nextCursor : undefined),
    staleTime: 10_000
  });

  const all = query.data?.pages.flatMap((p) => p.items) ?? [];
  const fresh = all.filter((c) => !c.lastMessage);
  const talking = all.filter((c) => c.lastMessage);

  const open = (c: Conversation) =>
    navigation.navigate("Chat", { matchId: c.matchId, partnerId: c.partner.userId, partnerName: c.partner.firstName });

  return (
    <Screen edges={["top"]} padded={false}>
      <View style={{ paddingHorizontal: spacing.lg, paddingVertical: spacing.sm }}>
        <AppText variant="heading">Messages</AppText>
      </View>
      {query.isLoading ? <Loading /> : null}
      {query.isError ? <ErrorState message={apiErrorMessage(query.error)} onRetry={() => void query.refetch()} /> : null}
      {!query.isLoading && !query.isError && all.length === 0 ? (
        <EmptyState
          emoji="💬"
          title="Pas encore de match"
          message="Quand une personne vous plaît autant que vous lui plaisez, la conversation s'ouvre ici."
        />
      ) : null}
      {all.length > 0 ? (
        <FlatList
          data={talking}
          keyExtractor={(c) => c.matchId}
          refreshControl={
            <RefreshControl
              refreshing={query.isRefetching && !query.isFetchingNextPage}
              onRefresh={() => void query.refetch()}
              tintColor={colors.primary}
            />
          }
          onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
          onEndReachedThreshold={0.4}
          ListHeaderComponent={
            fresh.length > 0 ? (
              <View style={{ paddingBottom: spacing.md }}>
                <AppText variant="label" muted>
                  Nouveaux matchs
                </AppText>
                <ScrollView
                  horizontal
                  showsHorizontalScrollIndicator={false}
                  contentContainerStyle={{ gap: spacing.lg, paddingVertical: spacing.md, paddingHorizontal: spacing.lg }}
                >
                  {fresh.map((c) => (
                    <Pressable
                      key={c.matchId}
                      accessibilityRole="button"
                      accessibilityLabel={`Nouveau match avec ${c.partner.firstName}`}
                      onPress={() => open(c)}
                      style={{ alignItems: "center", gap: 6 }}
                    >
                      <Avatar url={c.partner.photoUrl} name={c.partner.firstName} size={64} />
                      <Text style={{ color: colors.text, fontWeight: "600" }}>{c.partner.firstName}</Text>
                    </Pressable>
                  ))}
                </ScrollView>
                {talking.length > 0 ? (
                  <AppText variant="label" muted>
                    Conversations
                  </AppText>
                ) : null}
              </View>
            ) : null
          }
          contentContainerStyle={{ paddingHorizontal: spacing.lg }}
          renderItem={({ item }) => <ConversationRow item={item} mine={user?.id ?? ""} onPress={() => open(item)} />}
        />
      ) : null}
    </Screen>
  );
}

function ConversationRow({ item, mine, onPress }: { item: Conversation; mine: string; onPress: () => void }) {
  const { colors } = useTheme();
  const last = item.lastMessage;
  const unread = item.unreadCount > 0;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`Conversation avec ${item.partner.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
      onPress={onPress}
      style={({ pressed }) => ({
        flexDirection: "row",
        alignItems: "center",
        gap: spacing.md,
        paddingVertical: spacing.md,
        opacity: pressed ? 0.7 : 1
      })}
    >
      <Avatar url={item.partner.photoUrl} name={item.partner.firstName} size={56} />
      <View style={{ flex: 1, gap: 2 }}>
        <Text style={{ color: colors.text, fontWeight: unread ? "800" : "600", fontSize: 16 }}>{item.partner.firstName}</Text>
        <Text numberOfLines={1} style={{ color: unread ? colors.text : colors.textMuted, fontWeight: unread ? "600" : "400" }}>
          {last ? `${last.senderId === mine ? "Vous : " : ""}${last.body}` : ""}
        </Text>
      </View>
      <View style={{ alignItems: "flex-end", gap: 6 }}>
        <Text style={{ color: colors.textMuted, fontSize: 12 }}>{last ? formatActivity(last.createdAt) : ""}</Text>
        <Badge count={item.unreadCount} />
      </View>
    </Pressable>
  );
}
