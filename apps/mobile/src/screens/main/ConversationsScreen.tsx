import React from "react";
import { Alert, FlatList, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { hideConversation } from "../../api/social";
import type { Conversation } from "../../api/types";
import { useConversations } from "../../hooks/queries";
import { formatListDate } from "../../lib/dates";
import { errorMessage } from "../../lib/errors";
import { useAuth } from "../../auth/AuthProvider";
import type { MainStackParams, TabParams } from "../../navigation/types";
import { Avatar, EmptyState, ErrorState, Loading, Text, useToast } from "../../ui";
import { MIN_TOUCH, radius, spacing } from "../../theme/tokens";
import { useTheme } from "../../theme/ThemeProvider";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParams, "Messages">,
  NativeStackScreenProps<MainStackParams>
>;

export default function ConversationsScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { user } = useAuth();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useConversations();

  if (query.isPending) return <Loading />;
  if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;

  const all = query.data.pages.flatMap((p) => p.conversations);
  const fresh = all.filter((c) => !c.lastMessage);
  const chats = all.filter((c) => c.lastMessage);

  const open = (c: Conversation) =>
    navigation.navigate("Chat", { conversationId: c.id, name: c.profile.firstName });

  const confirmHide = (c: Conversation) =>
    Alert.alert(
      `Supprimer la conversation avec ${c.profile.firstName} ?`,
      "Elle disparaîtra de votre liste. Votre match est conservé et la conversation réapparaîtra à son prochain message.",
      [
        { text: "Annuler", style: "cancel" },
        {
          text: "Supprimer",
          style: "destructive",
          onPress: () => {
            hideConversation(c.id)
              .then(() => queryClient.invalidateQueries({ queryKey: ["conversations"] }))
              .catch((e) => toast.show(errorMessage(e), "error"));
          }
        }
      ]
    );

  if (all.length === 0) {
    return (
      <EmptyState
        icon="✉"
        title="Aucune conversation"
        message="Quand vous matcherez avec quelqu'un, la conversation apparaîtra ici."
        actionLabel="Découvrir des profils"
        onAction={() => navigation.navigate("Discover")}
      />
    );
  }

  return (
    <FlatList
      testID="conversations-list"
      data={chats}
      keyExtractor={(c) => c.id}
      refreshing={query.isRefetching}
      onRefresh={() => void query.refetch()}
      onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
      ListHeaderComponent={
        fresh.length > 0 ? (
          <View style={styles.freshWrap}>
            <Text variant="caption" tone="muted" bold style={styles.section}>
              Nouveaux matchs
            </Text>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={styles.fresh}
            >
              {fresh.map((c) => (
                <Pressable
                  key={c.id}
                  accessibilityRole="button"
                  accessibilityLabel={`Nouveau match avec ${c.profile.firstName}`}
                  style={styles.freshItem}
                  onPress={() => open(c)}
                  testID={`fresh-${c.id}`}
                >
                  <Avatar photoId={c.profile.photos[0]?.id} name={c.profile.firstName} size={64} />
                  <Text variant="caption" numberOfLines={1}>
                    {c.profile.firstName}
                  </Text>
                </Pressable>
              ))}
            </ScrollView>
          </View>
        ) : null
      }
      ListEmptyComponent={
        <Text tone="muted" style={styles.hint}>
          Dites bonjour à l&apos;un de vos nouveaux matchs 👋
        </Text>
      }
      renderItem={({ item }) => {
        const last = item.lastMessage;
        const mine = last?.senderId === user?.id;
        return (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`${item.profile.firstName}${item.unreadCount ? `, ${item.unreadCount} non lus` : ""}`}
            onPress={() => open(item)}
            onLongPress={() => confirmHide(item)}
            style={({ pressed }) => [styles.row, { opacity: pressed ? 0.8 : 1, borderColor: colors.border }]}
            testID={`conversation-${item.id}`}
          >
            <Avatar photoId={item.profile.photos[0]?.id} name={item.profile.firstName} />
            <View style={styles.flex}>
              <Text bold={item.unreadCount > 0} numberOfLines={1} variant="subtitle">
                {item.profile.firstName}
              </Text>
              <Text
                tone={item.unreadCount > 0 ? "default" : "muted"}
                bold={item.unreadCount > 0}
                numberOfLines={1}
              >
                {mine ? "Vous : " : ""}
                {last?.body}
              </Text>
            </View>
            <View style={styles.meta}>
              <Text variant="caption" tone="muted">
                {last ? formatListDate(last.createdAt) : ""}
              </Text>
              {item.unreadCount > 0 ? (
                <View style={[styles.badge, { backgroundColor: colors.accent }]} accessibilityElementsHidden>
                  <Text variant="caption" tone="onAccent" bold>
                    {item.unreadCount > 99 ? "99+" : item.unreadCount}
                  </Text>
                </View>
              ) : null}
            </View>
          </Pressable>
        );
      }}
    />
  );
}

const styles = StyleSheet.create({
  freshWrap: { paddingTop: spacing.md, gap: spacing.sm },
  section: { paddingHorizontal: spacing.lg },
  fresh: { paddingHorizontal: spacing.lg, gap: spacing.lg },
  freshItem: { alignItems: "center", gap: spacing.xs, width: 68 },
  hint: { padding: spacing.xl, textAlign: "center" },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    minHeight: MIN_TOUCH + 24,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  flex: { flex: 1 },
  meta: { alignItems: "flex-end", gap: spacing.xs },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: radius.pill,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6
  }
});
