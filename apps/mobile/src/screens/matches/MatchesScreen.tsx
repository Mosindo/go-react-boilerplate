import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import type { ChatSummary, Match } from "../../api/types";
import { useAuth } from "../../hooks/useAuth";
import { useChats, useMatches } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { Avatar, Badge, EmptyState, ErrorState, Loading, Photo, Screen, Text } from "../../ui";
import { relativeTime } from "../../utils/format";

export default function MatchesScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const { colors } = useTheme();
  const { user } = useAuth();
  const matches = useMatches();
  const chats = useChats();

  const fresh = useMemo(() => (matches.data ?? []).filter((m) => !m.lastMessageAt), [matches.data]);
  const open = (userId: string, firstName: string) => navigation.navigate("Chat", { userId, firstName });
  const refreshing = (matches.isRefetching || chats.isRefetching) && !matches.isLoading;

  if (matches.isLoading || chats.isLoading) return <Loading />;
  if (matches.error || chats.error) {
    return (
      <Screen>
        <ErrorState message={errorMessage(matches.error ?? chats.error)} onRetry={() => void Promise.all([matches.refetch(), chats.refetch()])} />
      </Screen>
    );
  }
  if (!fresh.length && !chats.data?.length) {
    return (
      <Screen>
        <Text variant="title" serif style={styles.title} accessibilityRole="header">
          Matchs
        </Text>
        <EmptyState icon="heart-outline" title="Pas encore de match" message="Quand l'intérêt est réciproque, vous pourrez discuter ici." />
      </Screen>
    );
  }

  return (
    <Screen padded={false}>
      <FlatList<ChatSummary>
        data={chats.data ?? []}
        keyExtractor={(c) => c.matchId}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={() => void Promise.all([matches.refetch(), chats.refetch()])} tintColor={colors.primary} />
        }
        ListHeaderComponent={
          <View>
            <Text variant="title" serif style={[styles.title, styles.padded]} accessibilityRole="header">
              Matchs
            </Text>
            {fresh.length ? (
              <View style={styles.section}>
                <Text variant="overline" tone="subtle" style={styles.padded}>
                  Nouveaux matchs
                </Text>
                <FlatList<Match>
                  horizontal
                  data={fresh}
                  keyExtractor={(m) => m.id}
                  showsHorizontalScrollIndicator={false}
                  contentContainerStyle={styles.freshList}
                  renderItem={({ item }) => (
                    <Pressable
                      accessibilityRole="button"
                      accessibilityLabel={`Écrire à ${item.user.firstName}`}
                      onPress={() => open(item.user.userId, item.user.firstName)}
                      style={styles.fresh}
                    >
                      <Photo url={item.user.photos[0]?.url} style={[styles.freshPhoto, { borderColor: colors.primary }]} />
                      <Text variant="label" numberOfLines={1}>
                        {item.user.firstName}
                      </Text>
                    </Pressable>
                  )}
                />
              </View>
            ) : null}
            {chats.data?.length ? (
              <Text variant="overline" tone="subtle" style={[styles.padded, styles.section]}>
                Messages
              </Text>
            ) : null}
          </View>
        }
        renderItem={({ item }) => {
          const mine = item.lastMessage?.senderUserId === user?.id;
          const unread = item.unreadCount > 0;
          return (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`Conversation avec ${item.user.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
              onPress={() => open(item.user.userId, item.user.firstName)}
              style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
              testID={`chat-row-${item.user.firstName}`}
            >
              <Avatar url={item.user.photos[0]?.url} size={56} />
              <View style={styles.rowText}>
                <View style={styles.rowTop}>
                  <Text variant="bodyStrong" numberOfLines={1} style={styles.flex}>
                    {item.user.firstName}
                  </Text>
                  {item.lastMessage ? (
                    <Text variant="caption" tone={unread ? "primary" : "subtle"}>
                      {relativeTime(item.lastMessage.createdAt)}
                    </Text>
                  ) : null}
                </View>
                <View style={styles.rowTop}>
                  <Text tone={unread ? "default" : "muted"} numberOfLines={1} style={[styles.flex, unread ? { fontWeight: "600" } : null]}>
                    {mine ? "Vous : " : ""}
                    {item.lastMessage?.content}
                  </Text>
                  <Badge count={item.unreadCount} />
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
  title: { paddingTop: spacing.xs, marginBottom: spacing.sm },
  padded: { paddingHorizontal: spacing.lg },
  section: { marginTop: spacing.sm, gap: spacing.xs },
  freshList: { paddingHorizontal: spacing.lg, gap: spacing.md, paddingVertical: spacing.xs },
  fresh: { width: 76, alignItems: "center", gap: 6 },
  freshPhoto: { width: 72, height: 92, borderRadius: 18, borderWidth: 2 },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  rowText: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  flex: { flex: 1 }
});
