import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import type { Conversation } from "../../api/types";
import { useConversations } from "../../hooks/queries";
import { useSession } from "../../hooks/useSession";
import type { AppStackParamList } from "../../navigation/types";
import { colors, radii, spacing } from "../../theme";
import { Avatar, EmptyState, ErrorState, Header, LoadingState, Screen, Text } from "../../ui";
import { formatListTime } from "../../utils/format";

export function MatchesScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const { user } = useSession();
  const conversations = useConversations();
  const all = useMemo(() => conversations.data?.pages.flatMap((p) => p.conversations) ?? [], [conversations.data]);
  const fresh = all.filter((c) => !c.lastMessage);
  const threads = all.filter((c) => c.lastMessage);
  const open = (c: Conversation) => navigation.navigate("Chat", { conversationId: c.id });

  if (conversations.isLoading) {
    return (
      <Screen edges={["top"]}>
        <Header title="Matchs" large />
        <LoadingState />
      </Screen>
    );
  }
  if (conversations.error) {
    return (
      <Screen edges={["top"]}>
        <Header title="Matchs" large />
        <ErrorState message={errorMessage(conversations.error)} onRetry={() => void conversations.refetch()} />
      </Screen>
    );
  }

  return (
    <Screen edges={["top"]} padded={false} testID="matches-screen">
      <View style={styles.header}>
        <Header title="Matchs" large />
      </View>
      {all.length === 0 ? (
        <EmptyState
          icon="heart-circle-outline"
          title="Pas encore de match"
          body="Quand une personne que vous aimez vous aime aussi, elle apparaîtra ici."
          actionLabel="Découvrir des profils"
          onAction={() => navigation.navigate("Tabs", { screen: "Discover" })}
          testID="matches-empty"
        />
      ) : (
        <FlatList
          data={threads}
          keyExtractor={(c) => c.id}
          refreshControl={
            <RefreshControl
              refreshing={conversations.isRefetching}
              onRefresh={() => void conversations.refetch()}
              tintColor={colors.primary}
            />
          }
          onEndReached={() => conversations.hasNextPage && void conversations.fetchNextPage()}
          contentContainerStyle={styles.list}
          ListHeaderComponent={
            fresh.length ? (
              <View style={styles.freshSection}>
                <Text variant="label" color="textSubtle" style={styles.sectionTitle}>
                  NOUVEAUX MATCHS
                </Text>
                <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.freshRow}>
                  {fresh.map((c) => (
                    <Pressable
                      key={c.id}
                      onPress={() => open(c)}
                      accessibilityRole="button"
                      accessibilityLabel={`Écrire à ${c.peer.firstName}`}
                      style={styles.fresh}
                      testID={`new-match-${c.peer.firstName}`}
                    >
                      <Avatar path={c.peer.photo?.url} name={c.peer.firstName} size={68} ring />
                      <Text variant="caption" numberOfLines={1}>
                        {c.peer.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
                {threads.length ? (
                  <Text variant="label" color="textSubtle" style={styles.sectionTitle}>
                    MESSAGES
                  </Text>
                ) : null}
              </View>
            ) : (
              <Text variant="label" color="textSubtle" style={styles.sectionTitle}>
                MESSAGES
              </Text>
            )
          }
          renderItem={({ item }) => {
            const mine = item.lastMessage?.senderId === user?.id;
            return (
              <Pressable
                onPress={() => open(item)}
                accessibilityRole="button"
                accessibilityLabel={`Conversation avec ${item.peer.firstName}${item.unreadCount ? `, ${item.unreadCount} non lus` : ""}`}
                style={({ pressed }) => [styles.row, pressed && styles.pressed]}
                testID={`conversation-${item.peer.firstName}`}
              >
                <Avatar path={item.peer.photo?.url} name={item.peer.firstName} size={56} />
                <View style={styles.rowText}>
                  <View style={styles.rowTop}>
                    <Text variant="bodyStrong" numberOfLines={1} style={styles.flex}>
                      {item.peer.firstName}
                    </Text>
                    {item.lastMessage ? (
                      <Text variant="caption" color="textSubtle">
                        {formatListTime(item.lastMessage.createdAt)}
                      </Text>
                    ) : null}
                  </View>
                  <View style={styles.rowTop}>
                    <Text variant="small" color={item.unreadCount ? "text" : "textMuted"} numberOfLines={1} style={styles.flex}>
                      {mine ? "Vous : " : ""}
                      {item.lastMessage?.body}
                    </Text>
                    {item.unreadCount ? (
                      <View style={styles.badge}>
                        <Text variant="caption" color="onPrimary">
                          {item.unreadCount > 99 ? "99+" : item.unreadCount}
                        </Text>
                      </View>
                    ) : null}
                  </View>
                </View>
              </Pressable>
            );
          }}
          ListEmptyComponent={
            <Text variant="small" color="textSubtle" style={styles.hint}>
              Envoyez le premier message à l’un de vos matchs !
            </Text>
          }
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md
  },
  list: {
    paddingBottom: spacing.xl
  },
  freshSection: {
    gap: spacing.xs
  },
  sectionTitle: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md,
    paddingBottom: spacing.xs
  },
  freshRow: {
    paddingHorizontal: spacing.lg,
    gap: spacing.md
  },
  fresh: {
    width: 76,
    alignItems: "center",
    gap: spacing.xxs
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm
  },
  pressed: {
    backgroundColor: colors.surface
  },
  rowText: {
    flex: 1,
    gap: 2
  },
  rowTop: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  flex: {
    flex: 1
  },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: radii.pill,
    paddingHorizontal: 6,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center"
  },
  hint: {
    paddingHorizontal: spacing.lg
  }
});
