import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { absoluteUrl } from "../api/client";
import type { Conversation } from "../api/types";
import { useConversations } from "../hooks/queries";
import { useRealtime } from "../hooks/useRealtime";
import { formatRelativeTime } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Avatar, Badge, Card, Text, colors, spacing } from "../shared/ui";

/** Messages tab: one row per match, most recent activity first. */
export default function ChatScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { connected } = useRealtime();
  const query = useConversations(connected);

  return (
    <ScreenContainer testID="screen-chat">
      <Header subtitle={connected ? "En direct" : "Reconnexion en cours…"} title="Messages" />
      {query.isLoading ? <LoadingView label="Chargement des conversations…" /> : null}
      {query.isError && !query.data ? (
        <ErrorView message="Impossible de charger vos conversations." onAction={() => void query.refetch()} />
      ) : null}
      {query.data ? (
        <FlatList<Conversation>
          contentContainerStyle={styles.list}
          data={query.data}
          keyExtractor={(item) => item.matchId}
          ListEmptyComponent={
            <EmptyView
              message="Likez des profils : dès qu'un intérêt est réciproque, la conversation s'ouvre ici."
              testID="chat-empty"
              title="Aucune conversation"
            />
          }
          refreshControl={
            <RefreshControl onRefresh={() => void query.refetch()} refreshing={query.isRefetching} tintColor={colors.primary} />
          }
          renderItem={({ item }) => (
            <ConversationRow
              conversation={item}
              onPress={() => navigation.navigate("Thread", { matchId: item.matchId, profile: item.profile })}
            />
          )}
          showsVerticalScrollIndicator={false}
        />
      ) : null}
    </ScreenContainer>
  );
}

function ConversationRow({ conversation, onPress }: { conversation: Conversation; onPress: () => void }) {
  const { profile, lastMessage, unreadCount } = conversation;
  const photo = profile.photos[0];
  const preview = lastMessage ? lastMessage.body : "Nouveau match — dites bonjour !";
  const unread = unreadCount > 0;
  return (
    <Pressable
      accessibilityLabel={`Conversation avec ${profile.firstName}${unread ? `, ${unreadCount} non lu` : ""}`}
      accessibilityRole="button"
      onPress={onPress}
      testID={`conversation-${conversation.matchId}`}
    >
      <Card padding="md" style={styles.row} variant={unread ? "accent" : "default"}>
        <Avatar name={profile.firstName} size={56} uri={photo ? absoluteUrl(photo.url) : null} />
        <View style={styles.copy}>
          <View style={styles.titleRow}>
            <Text variant="label" weight="bold">
              {profile.firstName}
            </Text>
            <Text tone="muted" variant="caption">
              {formatRelativeTime(lastMessage?.createdAt ?? conversation.createdAt)}
            </Text>
          </View>
          <Text numberOfLines={1} tone={unread ? "default" : "muted"} weight={unread ? "semibold" : "regular"}>
            {lastMessage && lastMessage.senderId !== profile.id ? `Vous : ${preview}` : preview}
          </Text>
        </View>
        {unread ? <Badge label={String(unreadCount)} size="sm" variant="primary" /> : null}
      </Card>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  list: { paddingTop: spacing.lg, paddingBottom: spacing.xxxl, gap: spacing.sm },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  copy: { flex: 1, gap: spacing.xxs },
  titleRow: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" }
});
