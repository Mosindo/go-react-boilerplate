import React from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation, type CompositeNavigationProp } from "@react-navigation/native";
import type { BottomTabNavigationProp } from "@react-navigation/bottom-tabs";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import type { ChatSummary } from "../../api/types";
import { useChats } from "../../hooks/queries";
import { useCurrentUser } from "../../hooks/useAuth";
import { formatListTime } from "../../lib/format";
import type { MainStackParamList, TabParamList } from "../../navigation/types";
import { Avatar, EmptyState, ErrorState, Header, LoadingState, Screen, Text, colors, radii, spacing } from "../../ui";

type Nav = CompositeNavigationProp<BottomTabNavigationProp<TabParamList, "Messages">, NativeStackNavigationProp<MainStackParamList>>;

export default function ChatsScreen() {
  const navigation = useNavigation<Nav>();
  const user = useCurrentUser();
  const chats = useChats();

  const open = (c: ChatSummary) =>
    navigation.navigate("ChatThread", { userId: c.user.id, firstName: c.user.firstName, photoUrl: c.user.photo?.url });

  if (chats.isPending) {
    return (
      <Screen>
        <Header title="Messages" />
        <LoadingState />
      </Screen>
    );
  }
  if (chats.isError) {
    return (
      <Screen>
        <Header title="Messages" />
        <ErrorState message={errorMessage(chats.error)} onAction={() => void chats.refetch()} />
      </Screen>
    );
  }

  const fresh = chats.data.chats.filter((c) => !c.lastMessage);
  const conversations = chats.data.chats.filter((c) => c.lastMessage);

  return (
    <Screen testID="chats-screen">
      <Header title="Messages" />
      {chats.data.chats.length === 0 ? (
        <EmptyState
          actionLabel="Découvrir des profils"
          icon="chatbubbles-outline"
          message="Quand un like est réciproque, vous pouvez discuter ici."
          onAction={() => navigation.navigate("Discover")}
          title="Pas encore de match"
        />
      ) : (
        <FlatList
          contentContainerStyle={styles.list}
          data={conversations}
          keyExtractor={(c) => c.matchId}
          ListHeaderComponent={
            fresh.length ? (
              <View style={styles.freshWrap}>
                <Text tone="subtle" variant="eyebrow">
                  Nouveaux matchs
                </Text>
                <ScrollView contentContainerStyle={styles.fresh} horizontal showsHorizontalScrollIndicator={false}>
                  {fresh.map((c) => (
                    <Pressable
                      accessibilityLabel={`Écrire à ${c.user.firstName}`}
                      accessibilityRole="button"
                      key={c.matchId}
                      onPress={() => open(c)}
                      style={styles.freshItem}
                      testID={`new-match-${c.user.id}`}
                    >
                      <Avatar name={c.user.firstName} ring size={68} url={c.user.photo?.url} />
                      <Text numberOfLines={1} variant="caption">
                        {c.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
                {conversations.length ? (
                  <Text tone="subtle" variant="eyebrow">
                    Conversations
                  </Text>
                ) : null}
              </View>
            ) : null
          }
          refreshControl={
            <RefreshControl onRefresh={() => void chats.refetch()} refreshing={chats.isRefetching} tintColor={colors.accent} />
          }
          renderItem={({ item }) => {
            const last = item.lastMessage!;
            const mine = last.senderUserId === user.id;
            const unread = item.unreadCount > 0;
            return (
              <Pressable
                accessibilityLabel={`Conversation avec ${item.user.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
                accessibilityRole="button"
                onPress={() => open(item)}
                style={({ pressed }) => [styles.row, pressed ? styles.pressed : null]}
                testID={`chat-${item.user.id}`}
              >
                <Avatar name={item.user.firstName} size={56} url={item.user.photo?.url} />
                <View style={styles.rowBody}>
                  <View style={styles.rowTop}>
                    <Text numberOfLines={1} style={styles.flex} variant="label">
                      {item.user.firstName}
                    </Text>
                    <Text tone={unread ? "accent" : "subtle"} variant="caption">
                      {formatListTime(last.createdAt)}
                    </Text>
                  </View>
                  <View style={styles.rowTop}>
                    <Text numberOfLines={1} style={styles.flex} tone={unread ? "default" : "muted"} variant="caption">
                      {mine ? "Vous : " : ""}
                      {last.content}
                    </Text>
                    {unread ? (
                      <View style={styles.badge}>
                        <Text tone="onAccent" variant="eyebrow">
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
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: {
    gap: spacing.xxs,
    paddingBottom: spacing.xl
  },
  freshWrap: {
    gap: spacing.sm,
    marginBottom: spacing.sm
  },
  fresh: {
    gap: spacing.md,
    paddingVertical: spacing.xs
  },
  freshItem: {
    width: 72,
    alignItems: "center",
    gap: spacing.xxs
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    padding: spacing.sm,
    borderRadius: radii.md
  },
  pressed: {
    backgroundColor: colors.surface
  },
  rowBody: {
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
    minWidth: 20,
    height: 20,
    paddingHorizontal: 6,
    borderRadius: 10,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.accent
  }
});
