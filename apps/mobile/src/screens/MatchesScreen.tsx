import React from "react";
import { FlatList, Pressable, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { mediaUrl } from "../api/client";
import type { Conversation } from "../api/types";
import { useConversations } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { formatListTime } from "../lib/dates";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Avatar, CountBadge, Text, spacing } from "../shared/ui";
import { useTheme } from "../theme";

export default function MatchesScreen() {
  const palette = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { user } = useAuth();
  const query = useConversations();
  const all = query.data ?? [];
  const fresh = all.filter((c) => !c.lastMessage);
  const talking = all.filter((c) => c.lastMessage);

  const open = (c: Conversation) =>
    navigation.navigate("Thread", { conversationId: c.id, peerId: c.peer.userId, peerName: c.peer.firstName });

  const header = (
    <View style={{ gap: spacing.lg, paddingBottom: spacing.md }}>
      <Text accessibilityRole="header" variant="heading" weight="bold">
        Matchs
      </Text>
      {fresh.length > 0 ? (
        <View style={{ gap: spacing.sm }}>
          <Text tone="muted" variant="eyebrow" weight="semibold">
            Nouveaux matchs
          </Text>
          <FlatList
            data={fresh}
            horizontal
            keyExtractor={(c) => c.id}
            renderItem={({ item }) => (
              <Pressable
                accessibilityLabel={`Nouveau match avec ${item.peer.firstName}. Écrire un premier message`}
                accessibilityRole="button"
                onPress={() => open(item)}
                style={{ alignItems: "center", gap: spacing.xs, width: 84 }}
                testID={`new-match-${item.peer.firstName}`}
              >
                <Avatar name={item.peer.firstName} ring size="lg" uri={mediaUrl(item.peer.thumbUrl)} />
                <Text numberOfLines={1} variant="label" weight="semibold">
                  {item.peer.firstName}
                </Text>
              </Pressable>
            )}
            showsHorizontalScrollIndicator={false}
          />
        </View>
      ) : null}
      {talking.length > 0 ? (
        <Text tone="muted" variant="eyebrow" weight="semibold">
          Messages
        </Text>
      ) : null}
    </View>
  );

  return (
    <ScreenContainer testID="matches-screen">
      {query.isLoading ? (
        <LoadingView fullScreen />
      ) : query.isError ? (
        <ErrorView error={query.error} onRetry={() => void query.refetch()} />
      ) : all.length === 0 ? (
        <View style={{ flex: 1 }}>
          <View style={{ minHeight: 56, justifyContent: "center" }}>
            <Text accessibilityRole="header" variant="heading" weight="bold">
              Matchs
            </Text>
          </View>
          <EmptyView
            icon={<Ionicons color={palette.textSubtle} name="chatbubbles-outline" size={56} />}
            message="Quand vous et une autre personne vous plairez mutuellement, votre conversation apparaîtra ici."
            style={{ flex: 1 }}
            testID="matches-empty"
            title="Pas encore de match"
          />
        </View>
      ) : (
        <FlatList
          data={talking}
          keyExtractor={(c) => c.id}
          ListHeaderComponent={header}
          onRefresh={() => void query.refetch()}
          refreshing={query.isRefetching}
          renderItem={({ item }) => {
            const last = item.lastMessage;
            const mine = last?.senderId === user?.id;
            return (
              <Pressable
                accessibilityLabel={`${item.peer.firstName}. ${last ? `${mine ? "Vous : " : ""}${last.body}` : ""}${item.unreadCount > 0 ? `. ${item.unreadCount} non lu` : ""}`}
                accessibilityRole="button"
                onPress={() => open(item)}
                style={({ pressed }) => ({
                  flexDirection: "row",
                  alignItems: "center",
                  gap: spacing.md,
                  paddingVertical: spacing.md,
                  opacity: pressed ? 0.7 : 1
                })}
                testID={`conversation-${item.peer.firstName}`}
              >
                <Avatar name={item.peer.firstName} uri={mediaUrl(item.peer.thumbUrl)} />
                <View style={{ flex: 1, gap: 2 }}>
                  <Text weight={item.unreadCount > 0 ? "bold" : "semibold"}>{item.peer.firstName}</Text>
                  <Text
                    numberOfLines={1}
                    tone={item.unreadCount > 0 ? "default" : "muted"}
                    variant="label"
                    weight={item.unreadCount > 0 ? "semibold" : "regular"}
                  >
                    {mine ? "Vous : " : ""}
                    {last?.body}
                  </Text>
                </View>
                <View style={{ alignItems: "flex-end", gap: spacing.xs }}>
                  <Text tone="subtle" variant="caption">
                    {formatListTime(item.updatedAt)}
                  </Text>
                  <CountBadge count={item.unreadCount} />
                </View>
              </Pressable>
            );
          }}
          showsVerticalScrollIndicator={false}
        />
      )}
    </ScreenContainer>
  );
}
