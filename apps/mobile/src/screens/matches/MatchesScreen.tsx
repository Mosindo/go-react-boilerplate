import React, { useCallback, useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useFocusEffect, useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { MatchSummary } from "../../api/types";
import { useAuth } from "../../hooks/useAuth";
import { useMatches } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../../shared/feedback";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Avatar, CountBadge, Text, colors, spacing } from "../../shared/ui";
import { formatRelative } from "../../utils/format";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export default function MatchesScreen() {
  const navigation = useNavigation<Nav>();
  const { user } = useAuth();
  const query = useMatches();
  const matches = useMemo(() => query.data?.pages.flatMap((p) => p.matches) ?? [], [query.data]);
  const fresh = matches.filter((m) => !m.lastMessage);
  const conversations = matches.filter((m) => m.lastMessage);
  const refetch = query.refetch;

  useFocusEffect(
    useCallback(() => {
      void refetch();
    }, [refetch])
  );

  const open = (m: MatchSummary) =>
    navigation.navigate("Chat", { conversationId: m.conversationId, matchId: m.id, userId: m.user.userId, firstName: m.user.firstName });

  if (query.isLoading) {
    return <LoadingView />;
  }
  if (query.isError) {
    return <ErrorView onRetry={() => void refetch()} />;
  }

  return (
    <Screen contentStyle={styles.content} scroll={false} testID="matches-screen">
      <View style={styles.header}>
        <ScreenHeader title="Matchs" />
      </View>
      {matches.length === 0 ? (
        <EmptyView
          actionLabel="Découvrir des profils"
          glyph="♥"
          message="Quand un like est réciproque, la personne apparaît ici et vous pouvez discuter."
          onAction={() => navigation.navigate("Tabs", { screen: "Discover" })}
          title="Pas encore de match"
        />
      ) : (
        <FlatList
          ListHeaderComponent={
            fresh.length > 0 ? (
              <View style={styles.freshSection}>
                <Text tone="muted" variant="eyebrow">
                  Nouveaux matchs
                </Text>
                <ScrollView contentContainerStyle={styles.freshRow} horizontal showsHorizontalScrollIndicator={false}>
                  {fresh.map((m) => (
                    <Pressable
                      accessibilityLabel={`Écrire à ${m.user.firstName}`}
                      key={m.id}
                      onPress={() => open(m)}
                      style={styles.freshItem}
                      testID={`new-match-${m.user.userId}`}
                    >
                      <Avatar name={m.user.firstName} size={72} url={m.user.photo?.url} />
                      <Text numberOfLines={1} variant="label" weight="medium">
                        {m.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
                {conversations.length > 0 ? (
                  <Text tone="muted" variant="eyebrow">
                    Messages
                  </Text>
                ) : null}
              </View>
            ) : null
          }
          contentContainerStyle={styles.list}
          data={conversations}
          keyExtractor={(m) => m.id}
          onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
          onEndReachedThreshold={0.4}
          refreshControl={<RefreshControl onRefresh={() => void refetch()} refreshing={query.isRefetching} tintColor={colors.primary} />}
          renderItem={({ item }) => {
            const mine = item.lastMessage?.senderUserId === user?.id;
            return (
              <Pressable
                accessibilityLabel={`Conversation avec ${item.user.firstName}`}
                onPress={() => open(item)}
                style={({ pressed }) => [styles.row, pressed ? styles.pressed : null]}
                testID={`conversation-${item.user.userId}`}
              >
                <Avatar name={item.user.firstName} size={56} url={item.user.photo?.url} />
                <View style={styles.rowText}>
                  <View style={styles.rowTop}>
                    <Text numberOfLines={1} style={styles.name} weight="semibold">
                      {item.user.firstName}
                    </Text>
                    {item.lastMessage ? (
                      <Text tone="subtle" variant="caption">
                        {formatRelative(item.lastMessage.createdAt)}
                      </Text>
                    ) : null}
                  </View>
                  <View style={styles.rowTop}>
                    <Text
                      numberOfLines={1}
                      style={styles.preview}
                      tone={item.unreadCount > 0 ? "default" : "muted"}
                      variant="label"
                      weight={item.unreadCount > 0 ? "semibold" : "regular"}
                    >
                      {mine ? "Toi : " : ""}
                      {item.lastMessage?.content}
                    </Text>
                    <CountBadge count={item.unreadCount} />
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
  content: {
    paddingHorizontal: 0,
    paddingBottom: 0
  },
  header: {
    paddingHorizontal: spacing.xl
  },
  list: {
    paddingHorizontal: spacing.xl,
    paddingBottom: spacing.xxxl
  },
  freshSection: {
    gap: spacing.md,
    paddingBottom: spacing.md
  },
  freshRow: {
    gap: spacing.lg
  },
  freshItem: {
    width: 76,
    alignItems: "center",
    gap: spacing.xs
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  pressed: {
    opacity: 0.7
  },
  rowText: {
    flex: 1,
    gap: 3
  },
  rowTop: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  },
  name: {
    flex: 1
  },
  preview: {
    flex: 1
  }
});
