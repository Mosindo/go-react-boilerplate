import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import type { MatchSummary } from "../api/types";
import { useMatches } from "../hooks/queries";
import { formatListTime } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import { radii, spacing, useTheme } from "../theme";
import { Avatar, EmptyState, ErrorState, LoadingState, Screen, Text } from "../ui";

export default function MessagesScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const { data, isLoading, isError, error, refetch, isRefetching } = useMatches();

  const open = (m: MatchSummary) => navigation.navigate("Chat", { matchId: m.id, userId: m.profile.userId, name: m.profile.firstName });

  if (isLoading) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (isError) {
    return (
      <Screen>
        <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />
      </Screen>
    );
  }

  const matches = data ?? [];
  const fresh = matches.filter((m) => !m.lastMessage);
  const conversations = matches.filter((m) => m.lastMessage);

  const header = (
    <View style={styles.header}>
      <Text variant="title">Messages</Text>
      {fresh.length > 0 ? (
        <View style={styles.rail}>
          <Text variant="label" tone="muted">
            Nouveaux matchs
          </Text>
          <FlatList
            horizontal
            showsHorizontalScrollIndicator={false}
            data={fresh}
            keyExtractor={(m) => m.id}
            contentContainerStyle={styles.railContent}
            renderItem={({ item }) => (
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={`Nouveau match avec ${item.profile.firstName}`}
                onPress={() => open(item)}
                style={styles.railItem}
              >
                <View style={[styles.ring, { borderColor: colors.primary }]}>
                  <Avatar uri={item.profile.photos[0]?.thumbUrl} name={item.profile.firstName} size={64} />
                </View>
                <Text variant="caption" numberOfLines={1}>
                  {item.profile.firstName}
                </Text>
              </Pressable>
            )}
          />
        </View>
      ) : null}
      {conversations.length > 0 ? (
        <Text variant="label" tone="muted">
          Conversations
        </Text>
      ) : null}
    </View>
  );

  return (
    <Screen padded={false}>
      <FlatList
        data={conversations}
        keyExtractor={(m) => m.id}
        ListHeaderComponent={header}
        ListEmptyComponent={
          matches.length === 0 ? (
            <EmptyState
              icon="chatbubbles-outline"
              title="Pas encore de match"
              message="Likez des profils : quand l'intérêt est réciproque, la conversation s'ouvre ici."
            />
          ) : (
            <Text tone="muted" style={styles.hint}>
              Dites bonjour à l'un de vos nouveaux matchs !
            </Text>
          )
        }
        refreshControl={<RefreshControl refreshing={isRefetching} onRefresh={() => void refetch()} tintColor={colors.primary} />}
        renderItem={({ item }) => {
          const last = item.lastMessage;
          const unread = item.unreadCount > 0;
          return (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`Conversation avec ${item.profile.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
              onPress={() => open(item)}
              style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
            >
              <Avatar uri={item.profile.photos[0]?.thumbUrl} name={item.profile.firstName} size={56} />
              <View style={styles.flex}>
                <View style={styles.rowTop}>
                  <Text variant="label" style={styles.flex} numberOfLines={1}>
                    {item.profile.firstName}
                  </Text>
                  {last ? (
                    <Text variant="caption" tone={unread ? "primary" : "muted"}>
                      {formatListTime(last.createdAt)}
                    </Text>
                  ) : null}
                </View>
                <View style={styles.rowTop}>
                  <Text tone={unread ? "default" : "muted"} numberOfLines={1} style={[styles.flex, unread ? styles.bold : null]}>
                    {last ? (last.senderId === item.profile.userId ? "" : "Vous : ") + last.body : ""}
                  </Text>
                  {unread ? (
                    <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                      <Text variant="caption" tone="onPrimary">
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

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { padding: spacing.lg, gap: spacing.lg },
  rail: { gap: spacing.sm },
  railContent: { gap: spacing.lg },
  railItem: { alignItems: "center", gap: spacing.xs, width: 72 },
  ring: { borderWidth: 2.5, borderRadius: 40, padding: 2 },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.md },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  bold: { fontWeight: "700" },
  badge: { minWidth: 22, height: 22, borderRadius: radii.pill, paddingHorizontal: 6, alignItems: "center", justifyContent: "center" },
  hint: { paddingHorizontal: spacing.lg }
});
