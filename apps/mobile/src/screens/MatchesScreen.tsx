import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { errorMessage, listConversations, queryKeys, type Conversation } from "../api/platform";
import { PhotoImage } from "../components/PhotoImage";
import { formatListTime } from "../lib/dates";
import type { RootStackParamList } from "../navigation/types";
import { SafeAreaLayout } from "../shared/layout";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Badge, Text, colors, radii, spacing } from "../shared/ui";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export default function MatchesScreen() {
  const navigation = useNavigation<Nav>();
  const query = useQuery({ queryKey: queryKeys.conversations, queryFn: listConversations });
  const all = query.data ?? [];
  const fresh = all.filter((c) => !c.lastMessage);
  const talking = all.filter((c) => c.lastMessage);

  const open = (c: Conversation) => navigation.navigate("Chat", { matchId: c.matchId, userId: c.user.id, name: c.user.firstName });

  if (query.isLoading) {
    return (
      <SafeAreaLayout edges={["top"]}>
        <LoadingView label="Loading your matches…" />
      </SafeAreaLayout>
    );
  }

  return (
    <SafeAreaLayout edges={["top", "left", "right"]}>
      <FlatList
        ListEmptyComponent={
          query.isError ? (
            <ErrorView message={errorMessage(query.error)} onAction={() => void query.refetch()} />
          ) : fresh.length === 0 ? (
            <EmptyView message="When you and someone else like each other, your conversation starts here." testID="matches-empty" title="No matches yet" />
          ) : null
        }
        ListHeaderComponent={
          <View style={styles.header}>
            <Text variant="title" weight="bold">
              Matches
            </Text>
            {fresh.length > 0 ? (
              <View style={styles.fresh}>
                <Text tone="muted" variant="eyebrow" weight="bold">
                  New matches
                </Text>
                <FlatList
                  data={fresh}
                  horizontal
                  keyExtractor={(c) => c.matchId}
                  renderItem={({ item }) => (
                    <Pressable accessibilityLabel={`New match ${item.user.firstName}`} accessibilityRole="button" onPress={() => open(item)} style={styles.freshItem} testID={`new-match-${item.user.firstName}`}>
                      <PhotoImage fallbackLabel={item.user.firstName} path={item.user.photoUrl} style={styles.freshPhoto} />
                      <Text numberOfLines={1} variant="caption" weight="semibold">
                        {item.user.firstName}
                      </Text>
                    </Pressable>
                  )}
                  showsHorizontalScrollIndicator={false}
                />
              </View>
            ) : null}
            {talking.length > 0 ? (
              <Text tone="muted" variant="eyebrow" weight="bold">
                Messages
              </Text>
            ) : null}
          </View>
        }
        contentContainerStyle={styles.list}
        data={talking}
        keyExtractor={(c) => c.matchId}
        onRefresh={() => void query.refetch()}
        refreshing={query.isRefetching}
        renderItem={({ item }) => {
          const mine = item.lastMessage?.senderId !== item.user.id;
          return (
            <Pressable
              accessibilityLabel={`${item.user.firstName}. ${item.unreadCount > 0 ? `${item.unreadCount} unread. ` : ""}${item.lastMessage?.body ?? ""}`}
              accessibilityRole="button"
              onPress={() => open(item)}
              style={styles.row}
              testID={`conversation-${item.user.firstName}`}
            >
              <PhotoImage fallbackLabel={item.user.firstName} path={item.user.photoUrl} style={styles.avatar} />
              <View style={styles.rowCopy}>
                <View style={styles.rowTop}>
                  <Text weight={item.unreadCount > 0 ? "bold" : "semibold"}>{item.user.firstName}</Text>
                  <Text tone="muted" variant="caption">
                    {item.lastMessage ? formatListTime(item.lastMessage.createdAt) : ""}
                  </Text>
                </View>
                <Text numberOfLines={1} tone={item.unreadCount > 0 ? "default" : "muted"} weight={item.unreadCount > 0 ? "semibold" : "regular"}>
                  {mine ? "You: " : ""}
                  {item.lastMessage?.body}
                </Text>
              </View>
              {item.unreadCount > 0 ? <Badge label={String(item.unreadCount)} variant="primary" /> : null}
            </Pressable>
          );
        }}
      />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, gap: spacing.sm, flexGrow: 1 },
  header: { gap: spacing.md, marginBottom: spacing.sm },
  fresh: { gap: spacing.sm },
  freshItem: { width: 76, marginRight: spacing.md, alignItems: "center", gap: spacing.xs },
  freshPhoto: { width: 68, height: 68, borderRadius: 34, borderWidth: 2, borderColor: colors.primary },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, padding: spacing.md, borderRadius: radii.lg, backgroundColor: colors.backgroundElevated, borderWidth: 1, borderColor: colors.border },
  avatar: { width: 56, height: 56, borderRadius: 28 },
  rowCopy: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", justifyContent: "space-between" }
});
