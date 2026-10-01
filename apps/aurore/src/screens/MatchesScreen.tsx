import React from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { chatApi, discoverApi } from "../api/endpoints";
import { keys } from "../api/keys";
import type { Conversation } from "../api/types";
import { Avatar } from "../components/AuthImage";
import { useFeedback } from "../components/Feedback";
import { errorMessage } from "../components/forms";
import { Badge, EmptyState, ErrorState, Loading, Screen, Separator, Text } from "../components/ui";
import { formatMessageTime } from "../lib/dates";
import { truncate } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme/theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export function MatchesScreen() {
  const t = useTheme();
  const nav = useNavigation<Nav>();
  const qc = useQueryClient();
  const { toast, confirm } = useFeedback();
  const matches = useQuery({ queryKey: keys.matches, queryFn: discoverApi.matches });
  const conversations = useQuery({ queryKey: keys.conversations, queryFn: chatApi.conversations });

  const hide = useMutation({
    mutationFn: chatApi.hide,
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.conversations }),
    onError: (e) => toast(errorMessage(e), "error")
  });

  const open = (c: Conversation) => nav.navigate("Chat", { conversationId: c.id, userId: c.user.userId, name: c.user.firstName });

  if (matches.isLoading || conversations.isLoading) {
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  }
  if (matches.error || conversations.error) {
    const err = (matches.error ?? conversations.error) as Error;
    return (
      <Screen>
        <ErrorState
          message={err.message}
          onRetry={() => {
            void matches.refetch();
            void conversations.refetch();
          }}
        />
      </Screen>
    );
  }

  const convs = conversations.data ?? [];
  const fresh = (matches.data ?? []).filter((m) => !convs.some((c) => c.id === m.conversationId && c.lastMessage));
  const refreshing = matches.isRefetching || conversations.isRefetching;

  const askHide = async (c: Conversation) => {
    const ok = await confirm({
      title: `Supprimer la conversation avec ${c.user.firstName} ?`,
      message: "Elle disparaît de votre liste seulement. Vous resterez en match.",
      confirmLabel: "Supprimer",
      destructive: true
    });
    if (ok) hide.mutate(c.id);
  };

  return (
    <Screen padded={false}>
      <FlatList
        data={convs}
        keyExtractor={(c) => c.id}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={() => {
              void matches.refetch();
              void conversations.refetch();
            }}
          />
        }
        ListHeaderComponent={
          <View>
            <Text variant="title" style={{ paddingHorizontal: spacing.lg, paddingTop: spacing.sm, paddingBottom: spacing.md }}>
              Matchs
            </Text>
            {fresh.length > 0 ? (
              <View style={{ marginBottom: spacing.lg }}>
                <Text variant="label" muted style={{ paddingHorizontal: spacing.lg, marginBottom: spacing.sm }}>
                  Nouveaux matchs
                </Text>
                <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={{ paddingHorizontal: spacing.lg, gap: spacing.lg }}>
                  {fresh.map((m) => (
                    <Pressable
                      key={m.id}
                      accessibilityRole="button"
                      accessibilityLabel={`Nouveau match : ${m.user.firstName}`}
                      testID="new-match"
                      onPress={() => nav.navigate("Chat", { conversationId: m.conversationId, userId: m.user.userId, name: m.user.firstName })}
                      style={{ alignItems: "center", width: 72 }}
                    >
                      <View style={{ borderWidth: 2, borderColor: t.primary, borderRadius: 36, padding: 2 }}>
                        <Avatar path={m.user.photos[0]?.thumbUrl} size={64} />
                      </View>
                      <Text variant="caption" numberOfLines={1} style={{ marginTop: 4 }}>
                        {m.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
              </View>
            ) : null}
            {convs.length > 0 ? (
              <Text variant="label" muted style={{ paddingHorizontal: spacing.lg, marginBottom: spacing.sm }}>
                Messages
              </Text>
            ) : null}
          </View>
        }
        renderItem={({ item }) => (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`Conversation avec ${item.user.firstName}${item.unreadCount ? `, ${item.unreadCount} non lus` : ""}`}
            testID="conversation-row"
            onPress={() => open(item)}
            onLongPress={() => void askHide(item)}
            style={({ pressed }) => ({ flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.md, backgroundColor: pressed ? t.surfaceAlt : "transparent" })}
          >
            <Avatar path={item.user.photos[0]?.thumbUrl} size={56} />
            <View style={{ flex: 1 }}>
              <View style={{ flexDirection: "row", justifyContent: "space-between" }}>
                <Text variant="heading" numberOfLines={1} style={{ flex: 1 }}>
                  {item.user.firstName}
                </Text>
                <Text variant="caption" muted>
                  {item.lastMessage ? formatMessageTime(item.lastMessage.createdAt) : ""}
                </Text>
              </View>
              <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.sm }}>
                <Text muted numberOfLines={1} style={{ flex: 1, fontWeight: item.unreadCount ? "700" : "400" }}>
                  {item.lastMessage ? truncate(item.lastMessage.body, 80) : "Nouveau match : dites bonjour !"}
                </Text>
                <Badge count={item.unreadCount} />
              </View>
            </View>
          </Pressable>
        )}
        ItemSeparatorComponent={Separator}
        ListEmptyComponent={
          fresh.length === 0 ? (
            <EmptyState icon="heart-outline" title="Pas encore de match" message="Likez des profils dans Découvrir : dès que l'intérêt est réciproque, la conversation s'ouvre ici." />
          ) : null
        }
        contentContainerStyle={{ flexGrow: 1 }}
      />
    </Screen>
  );
}
