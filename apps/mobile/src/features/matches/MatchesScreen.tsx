import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import type { ChatSummary } from "../../api/types";
import type { RootStackParamList, TabParamList } from "../../navigation/types";
import { makeStyles, useTheme } from "../../theme";
import { Avatar, EmptyState, ErrorState, LoadingState, Screen, Text } from "../../ui";
import { formatListTime } from "../../utils/format";
import { useAuth } from "../auth/AuthProvider";
import { useChats } from "./useChats";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Matches">, NativeStackScreenProps<RootStackParamList>>;

export function MatchesScreen({ navigation }: Props) {
  const styles = useStyles();
  const theme = useTheme();
  const { user } = useAuth();
  const chats = useChats();
  const all = useMemo(() => chats.data?.pages.flat() ?? [], [chats.data]);
  const fresh = all.filter((c) => !c.lastMessage);
  const conversations = all.filter((c) => c.lastMessage);

  const open = (chat: ChatSummary) =>
    navigation.navigate("Conversation", { userId: chat.user.id, firstName: chat.user.firstName, photoUrl: chat.user.photoUrl });

  if (chats.isLoading) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (chats.isError) {
    return (
      <Screen>
        <ErrorState message={errorMessage(chats.error)} onRetry={() => void chats.refetch()} />
      </Screen>
    );
  }

  return (
    <Screen padded={false}>
      <FlatList
        data={conversations}
        keyExtractor={(c) => c.matchId}
        refreshControl={
          <RefreshControl refreshing={chats.isRefetching} onRefresh={() => void chats.refetch()} tintColor={theme.colors.primary} />
        }
        onEndReached={() => {
          if (chats.hasNextPage && !chats.isFetchingNextPage) void chats.fetchNextPage();
        }}
        onEndReachedThreshold={0.4}
        contentContainerStyle={styles.list}
        ListHeaderComponent={
          <View style={styles.header}>
            <Text variant="title" accessibilityRole="header">
              Matchs
            </Text>
            {fresh.length > 0 ? (
              <View style={styles.freshBlock}>
                <Text variant="micro" tone="subtle">
                  NOUVEAUX MATCHS
                </Text>
                <FlatList
                  horizontal
                  data={fresh}
                  keyExtractor={(c) => c.matchId}
                  showsHorizontalScrollIndicator={false}
                  contentContainerStyle={styles.freshList}
                  renderItem={({ item }) => (
                    <Pressable
                      testID={`new-match-${item.user.id}`}
                      accessibilityRole="button"
                      accessibilityLabel={`Écrire à ${item.user.firstName}`}
                      onPress={() => open(item)}
                      style={styles.freshItem}
                    >
                      <Avatar uri={item.user.photoUrl} name={item.user.firstName} size={72} ring />
                      <Text variant="label" numberOfLines={1}>
                        {item.user.firstName}
                      </Text>
                    </Pressable>
                  )}
                />
              </View>
            ) : null}
            {conversations.length > 0 ? (
              <Text variant="micro" tone="subtle">
                MESSAGES
              </Text>
            ) : null}
          </View>
        }
        ListEmptyComponent={
          fresh.length === 0 ? (
            <EmptyState
              icon="heart-circle-outline"
              title="Pas encore de match"
              message="Quand deux personnes se likent mutuellement, la conversation s'ouvre ici."
              actionLabel="Découvrir des profils"
              onAction={() => navigation.navigate("Discover")}
            />
          ) : (
            <Text tone="muted" style={styles.hint}>
              Envoyez le premier message à vos nouveaux matchs !
            </Text>
          )
        }
        renderItem={({ item }) => {
          const mine = item.lastMessage?.senderUserId === user?.id;
          const unread = item.unreadCount > 0;
          return (
            <Pressable
              testID={`chat-row-${item.user.id}`}
              accessibilityRole="button"
              accessibilityLabel={`Conversation avec ${item.user.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
              onPress={() => open(item)}
              style={({ pressed }) => [styles.row, pressed ? styles.pressed : null]}
            >
              <Avatar uri={item.user.photoUrl} name={item.user.firstName} size={56} />
              <View style={styles.rowText}>
                <View style={styles.rowTop}>
                  <Text variant="bodyStrong" numberOfLines={1} style={styles.flex}>
                    {item.user.firstName}
                  </Text>
                  {item.lastMessage ? (
                    <Text variant="caption" tone={unread ? "primary" : "subtle"}>
                      {formatListTime(item.lastMessage.createdAt)}
                    </Text>
                  ) : null}
                </View>
                <View style={styles.rowTop}>
                  <Text
                    variant="caption"
                    tone={unread ? "default" : "muted"}
                    numberOfLines={1}
                    style={[styles.flex, unread ? styles.bold : null]}
                  >
                    {mine ? "Vous : " : ""}
                    {item.lastMessage?.content}
                  </Text>
                  {unread ? (
                    <View style={styles.badge}>
                      <Text variant="micro" tone="onPrimary">
                        {item.unreadCount > 99 ? "99+" : item.unreadCount}
                      </Text>
                    </View>
                  ) : null}
                </View>
              </View>
            </Pressable>
          );
        }}
      />
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  list: { paddingBottom: t.spacing.xxxl, flexGrow: 1 },
  header: { paddingHorizontal: t.spacing.xl, paddingTop: t.spacing.lg, gap: t.spacing.md, paddingBottom: t.spacing.sm },
  freshBlock: { gap: t.spacing.sm },
  freshList: { gap: t.spacing.lg, paddingVertical: t.spacing.xs },
  freshItem: { alignItems: "center", gap: t.spacing.xs, width: 76 },
  hint: { paddingHorizontal: t.spacing.xl },
  row: { flexDirection: "row", alignItems: "center", gap: t.spacing.md, paddingHorizontal: t.spacing.xl, paddingVertical: t.spacing.md },
  pressed: { backgroundColor: t.colors.surfaceAlt },
  rowText: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: t.spacing.sm },
  flex: { flex: 1 },
  bold: { fontWeight: "600" },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: 11,
    paddingHorizontal: 6,
    backgroundColor: t.colors.primary,
    alignItems: "center",
    justifyContent: "center"
  }
}));
