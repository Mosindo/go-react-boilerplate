import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { Match } from "../../api/types";
import { useAuth } from "../../hooks/useAuth";
import { useMatches } from "../../hooks/queries";
import type { MainStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, EmptyState, ErrorState, Header, LoadingState, Screen, Text } from "../../ui";
import { formatRelative } from "../../utils/format";

export function MatchesScreen() {
  const theme = useTheme();
  const { user } = useAuth();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const query = useMatches();
  const all = useMemo(() => query.data?.pages.flatMap((p) => p.items) ?? [], [query.data]);
  const fresh = all.filter((m) => !m.lastMessage);
  const conversations = all.filter((m) => m.lastMessage);
  const open = (m: Match) => navigation.navigate("Chat", { matchId: m.id, name: m.profile.firstName });

  if (query.isLoading)
    return (
      <Screen>
        <Header large title="Messages" />
        <LoadingState />
      </Screen>
    );
  if (query.error)
    return (
      <Screen>
        <Header large title="Messages" />
        <ErrorState onRetry={() => query.refetch()} />
      </Screen>
    );

  return (
    <Screen padded={false}>
      <View style={styles.pad}>
        <Header large title="Messages" />
      </View>
      {all.length === 0 ? (
        <EmptyState
          icon="chatbubbles-outline"
          message="Quand vous plairez à quelqu'un qui vous plaît, vous pourrez discuter ici."
          title="Pas encore de match"
        />
      ) : (
        <FlatList
          ListHeaderComponent={
            fresh.length ? (
              <View style={styles.freshWrap}>
                <Text style={styles.pad} tone="muted" variant="label">
                  NOUVEAUX MATCHS
                </Text>
                <ScrollView contentContainerStyle={styles.fresh} horizontal showsHorizontalScrollIndicator={false}>
                  {fresh.map((m) => (
                    <Pressable
                      accessibilityLabel={`Écrire à ${m.profile.firstName}`}
                      key={m.id}
                      onPress={() => open(m)}
                      style={styles.freshItem}
                      testID={`new-match-${m.profile.firstName}`}
                    >
                      <Avatar name={m.profile.firstName} photo={m.profile.photos[0]} ring size={72} />
                      <Text numberOfLines={1} variant="small">
                        {m.profile.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
                {conversations.length ? (
                  <Text style={[styles.pad, styles.convTitle]} tone="muted" variant="label">
                    CONVERSATIONS
                  </Text>
                ) : null}
              </View>
            ) : null
          }
          data={conversations}
          keyExtractor={(m) => m.id}
          onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && query.fetchNextPage()}
          onEndReachedThreshold={0.4}
          refreshControl={<RefreshControl onRefresh={() => query.refetch()} refreshing={query.isRefetching} tintColor={theme.colors.primary} />}
          renderItem={({ item }) => {
            const mine = item.lastMessage?.senderId === user?.id;
            const unread = item.unreadCount > 0;
            return (
              <Pressable
                accessibilityLabel={`Conversation avec ${item.profile.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
                onPress={() => open(item)}
                style={({ pressed }) => [styles.row, pressed && { backgroundColor: theme.colors.surfaceAlt }]}
                testID={`conversation-${item.profile.firstName}`}
              >
                <Avatar name={item.profile.firstName} photo={item.profile.photos[0]} size={56} />
                <View style={styles.rowBody}>
                  <View style={styles.rowTop}>
                    <Text style={styles.flex} variant="heading">
                      {item.profile.firstName}
                    </Text>
                    <Text tone="subtle" variant="caption">
                      {item.lastMessage ? formatRelative(item.lastMessage.createdAt) : ""}
                    </Text>
                  </View>
                  <View style={styles.rowTop}>
                    <Text numberOfLines={1} style={styles.flex} tone={unread ? "default" : "muted"} variant="small" weight={unread ? "600" : undefined}>
                      {mine ? "Vous : " : ""}
                      {item.lastMessage?.body}
                    </Text>
                    {unread ? (
                      <View style={[styles.badge, { backgroundColor: theme.colors.primary }]}>
                        <Text tone="onPrimary" variant="caption">
                          {item.unreadCount > 9 ? "9+" : item.unreadCount}
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
  pad: { paddingHorizontal: 20 },
  freshWrap: { gap: 12, paddingBottom: 8 },
  fresh: { gap: 14, paddingHorizontal: 20 },
  freshItem: { alignItems: "center", gap: 6, width: 76 },
  convTitle: { marginTop: 12 },
  row: { flexDirection: "row", alignItems: "center", gap: 14, paddingHorizontal: 20, paddingVertical: 12 },
  rowBody: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: 8 },
  flex: { flex: 1 },
  badge: { minWidth: 20, height: 20, borderRadius: 10, alignItems: "center", justifyContent: "center", paddingHorizontal: 6 }
});
