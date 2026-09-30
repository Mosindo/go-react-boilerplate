import React from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { Avatar, EmptyState, ErrorState, LoadingState, Screen, Text } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { errorMessage } from "../../lib/api/client";
import type { Conversation } from "../../lib/api/types";
import { formatRelative } from "../../lib/format";
import { useSession } from "../../lib/session/SessionProvider";
import type { AppStackParamList } from "../../navigation/types";
import { useConversations } from "./hooks";

export default function ConversationsScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const { colors } = useTheme();
  const { userId } = useSession();
  const query = useConversations();
  const all = query.data?.pages.flatMap((p) => p.conversations) ?? [];
  const fresh = all.filter((c) => !c.lastMessage);
  const threads = all.filter((c) => c.lastMessage);

  const open = (c: Conversation) => navigation.navigate("Chat", { conversationId: c.id, name: c.user.firstName });

  if (query.isLoading) return <LoadingState />;
  if (query.error && !query.data) return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;

  return (
    <Screen testID="conversations-screen">
      <Text style={styles.title} variant="title">
        Messages
      </Text>
      {all.length === 0 ? (
        <EmptyState icon="heart-outline" message="Quand un intérêt est réciproque, votre match apparaîtra ici et vous pourrez discuter." title="Pas encore de match" />
      ) : (
        <FlatList
          ListHeaderComponent={
            fresh.length ? (
              <View style={styles.freshSection}>
                <Text style={styles.sectionLabel} tone="muted" variant="overline">
                  NOUVEAUX MATCHS
                </Text>
                <ScrollView contentContainerStyle={styles.freshRow} horizontal showsHorizontalScrollIndicator={false}>
                  {fresh.map((c) => (
                    <Pressable accessibilityLabel={`Écrire à ${c.user.firstName}`} key={c.id} onPress={() => open(c)} style={styles.freshItem} testID={`new-match-${c.user.userId}`}>
                      <Avatar name={c.user.firstName} ring size={72} uri={c.user.photo?.url} />
                      <Text numberOfLines={1} variant="label">
                        {c.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
                {threads.length ? (
                  <Text style={styles.sectionLabel} tone="muted" variant="overline">
                    CONVERSATIONS
                  </Text>
                ) : null}
              </View>
            ) : null
          }
          data={threads}
          keyExtractor={(c) => c.id}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          refreshControl={<RefreshControl onRefresh={() => void query.refetch()} refreshing={query.isRefetching} tintColor={colors.primary} />}
          renderItem={({ item }) => {
            const unread = item.unreadCount > 0;
            const mine = item.lastMessage?.senderId === userId;
            return (
              <Pressable
                accessibilityLabel={`Conversation avec ${item.user.firstName}${unread ? `, ${item.unreadCount} non lu` : ""}`}
                onPress={() => open(item)}
                style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
                testID={`conversation-${item.user.userId}`}
              >
                <Avatar name={item.user.firstName} size={56} uri={item.user.photo?.url} />
                <View style={styles.rowText}>
                  <View style={styles.rowTop}>
                    <Text style={styles.flex} variant="label">
                      {item.user.firstName}
                    </Text>
                    {item.lastMessage ? (
                      <Text tone="subtle" variant="caption">
                        {formatRelative(item.lastMessage.createdAt)}
                      </Text>
                    ) : null}
                  </View>
                  <View style={styles.rowTop}>
                    <Text numberOfLines={1} style={[styles.flex, unread && styles.bold]} tone={unread ? "default" : "muted"} variant="caption">
                      {mine ? "Vous : " : ""}
                      {item.lastMessage?.body}
                    </Text>
                    {unread ? (
                      <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                        <Text style={{ color: colors.onPrimary }} variant="overline">
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
  title: { paddingHorizontal: 20, paddingVertical: 8 },
  freshSection: { gap: 10, paddingBottom: 8 },
  sectionLabel: { paddingHorizontal: 20 },
  freshRow: { paddingHorizontal: 16, gap: 14 },
  freshItem: { alignItems: "center", gap: 6, width: 76 },
  row: { flexDirection: "row", alignItems: "center", gap: 14, paddingHorizontal: 20, paddingVertical: 12 },
  rowText: { flex: 1, gap: 4 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: 8 },
  flex: { flex: 1 },
  bold: { fontWeight: "600" },
  badge: { minWidth: 22, height: 22, borderRadius: 11, alignItems: "center", justifyContent: "center", paddingHorizontal: 6 }
});
