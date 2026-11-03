import React from "react";
import { FlatList, Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { api } from "../api";
import { errorMessage } from "../api/client";
import type { Conversation } from "../api/types";
import { formatActivity } from "../lib/format";
import type { MainStackParams } from "../navigation/types";
import { radius, spacing, useTheme } from "../theme";
import { AuthImage } from "../ui/AuthImage";
import { Badge, EmptyState, ErrorState, Loading, Title } from "../ui/components";

export function useConversations() {
  return useQuery({ queryKey: ["conversations"], queryFn: api.conversations });
}

export function MessagesScreen() {
  const theme = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const conversations = useConversations();
  const open = (c: Conversation) => navigation.navigate("Chat", { conversationId: c.id, matchId: c.matchId, user: c.user });

  if (conversations.isLoading) {
    return <Loading />;
  }
  if (conversations.isError) {
    return <ErrorState message={errorMessage(conversations.error)} onRetry={() => void conversations.refetch()} />;
  }
  const all = conversations.data ?? [];
  const fresh = all.filter((c) => !c.lastMessage);
  const active = all.filter((c) => c.lastMessage);

  return (
    <View style={[styles.flex, { backgroundColor: theme.background }]}>
      <View style={styles.header}>
        <Title>Messages</Title>
      </View>
      {all.length === 0 ? (
        <EmptyState title="Aucune conversation" message="Quand un match se produit, vous pouvez discuter ici." />
      ) : (
        <FlatList
          data={active}
          keyExtractor={(c) => c.id}
          refreshing={conversations.isRefetching}
          onRefresh={() => void conversations.refetch()}
          ListHeaderComponent={
            fresh.length > 0 ? (
              <View style={styles.fresh}>
                <Text style={[styles.section, { color: theme.textMuted }]}>Nouveaux matchs</Text>
                <ScrollView
                  horizontal
                  showsHorizontalScrollIndicator={false}
                  contentContainerStyle={{ gap: spacing.lg, paddingHorizontal: spacing.xl }}
                >
                  {fresh.map((c) => (
                    <Pressable
                      key={c.id}
                      accessibilityRole="button"
                      accessibilityLabel={`Nouveau match : ${c.user.firstName}`}
                      onPress={() => open(c)}
                      style={styles.freshItem}
                      testID={`new-match-${c.user.firstName}`}
                    >
                      <AuthImage path={c.user.photos[0]?.url} style={styles.freshPhoto} />
                      <Text style={{ color: theme.text, fontWeight: "600" }} numberOfLines={1}>
                        {c.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
              </View>
            ) : null
          }
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              onPress={() => open(item)}
              style={[styles.row, { borderBottomColor: theme.border }]}
              testID={`conversation-${item.user.firstName}`}
            >
              <AuthImage path={item.user.photos[0]?.url} style={styles.avatar} />
              <View style={styles.flex}>
                <Text style={{ color: theme.text, fontSize: 17, fontWeight: item.unreadCount > 0 ? "800" : "600" }}>
                  {item.user.firstName}
                </Text>
                <Text
                  numberOfLines={1}
                  style={{ color: item.unreadCount > 0 ? theme.text : theme.textMuted, fontWeight: item.unreadCount > 0 ? "700" : "400" }}
                >
                  {item.lastMessage ? `${item.lastMessage.senderId === item.user.id ? "" : "Vous : "}${item.lastMessage.body}` : ""}
                </Text>
              </View>
              <View style={{ alignItems: "flex-end", gap: spacing.xs }}>
                <Text style={{ color: theme.textMuted, fontSize: 12 }}>{formatActivity(item.activityAt)}</Text>
                <Badge count={item.unreadCount} />
              </View>
            </Pressable>
          )}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { padding: spacing.xl, paddingBottom: spacing.sm },
  fresh: { paddingBottom: spacing.lg, gap: spacing.sm },
  section: { paddingHorizontal: spacing.xl, fontSize: 13, fontWeight: "700", textTransform: "uppercase" },
  freshItem: { alignItems: "center", gap: spacing.xs, width: 76 },
  freshPhoto: { width: 68, height: 68, borderRadius: 34 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.xl,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  avatar: { width: 56, height: 56, borderRadius: radius.pill }
});
