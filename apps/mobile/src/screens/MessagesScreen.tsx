import React, { useState } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { hideConversation } from "../api/platform";
import type { Conversation } from "../api/types";
import { useConversations, useMatches } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { formatListTime } from "../lib/format";
import { queryKeys } from "../lib/queryClient";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { ActionSheet, Avatar, Badge, ListItem, Text, spacing } from "../shared/ui";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export default function MessagesScreen() {
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const matches = useMatches();
  const conversations = useConversations();
  const [selected, setSelected] = useState<Conversation | null>(null);

  const hide = useMutation({
    mutationFn: (id: string) => hideConversation(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.conversations })
  });

  const fresh = (matches.data ?? []).filter((m) => m.lastMessageAt === null);
  const chats = conversations.data ?? [];
  const refreshing = matches.isRefetching || conversations.isRefetching;
  const refresh = () => {
    void matches.refetch();
    void conversations.refetch();
  };

  if (conversations.isLoading && matches.isLoading) {
    return (
      <ScreenContainer>
        <LoadingView label="Chargement…" />
      </ScreenContainer>
    );
  }
  if (conversations.isError && matches.isError) {
    return (
      <ScreenContainer>
        <ErrorView message="Impossible de charger vos conversations." onAction={refresh} />
      </ScreenContainer>
    );
  }

  const header = (
    <View style={styles.header}>
      <Header title="Messages" />
      {fresh.length > 0 ? (
        <View style={styles.rail}>
          <Text tone="muted" variant="eyebrow" weight="bold">
            Nouveaux matchs
          </Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.railContent}>
            {fresh.map((m) => (
              <Pressable
                accessibilityLabel={`Nouveau match avec ${m.profile.firstName}`}
                accessibilityRole="button"
                key={m.id}
                onPress={() =>
                  navigation.navigate("Conversation", {
                    conversationId: m.conversationId,
                    userId: m.profile.id,
                    name: m.profile.firstName,
                    photoId: m.profile.photos[0]?.id ?? null
                  })
                }
                style={styles.railItem}
                testID={`new-match-${m.profile.firstName}`}
              >
                <Avatar name={m.profile.firstName} photoId={m.profile.photos[0]?.id} size="lg" />
                <Text numberOfLines={1} variant="label" weight="semibold">
                  {m.profile.firstName}
                </Text>
              </Pressable>
            ))}
          </ScrollView>
        </View>
      ) : null}
      {chats.length > 0 ? (
        <Text tone="muted" variant="eyebrow" weight="bold">
          Conversations
        </Text>
      ) : null}
    </View>
  );

  return (
    <ScreenContainer>
      <FlatList
        ListEmptyComponent={
          fresh.length === 0 ? (
            <EmptyView
              icon="chatbubbles-outline"
              message="Likez des profils : dès qu'un intérêt est réciproque, vous pourrez discuter ici."
              title="Pas encore de match"
            />
          ) : (
            <Text style={styles.hint} tone="muted">
              Dites bonjour à l'un de vos nouveaux matchs 👋
            </Text>
          )
        }
        ListHeaderComponent={header}
        contentContainerStyle={styles.list}
        data={chats}
        keyExtractor={(item) => item.id}
        refreshControl={<RefreshControl onRefresh={refresh} refreshing={refreshing} />}
        renderItem={({ item }) => {
          const mine = item.lastMessage.senderId === user?.id;
          return (
            <ListItem
              leading={<Avatar name={item.profile.firstName} photoId={item.profile.photos[0]?.id} />}
              meta={formatListTime(item.lastMessage.createdAt)}
              onLongPress={() => setSelected(item)}
              onPress={() =>
                navigation.navigate("Conversation", {
                  conversationId: item.id,
                  userId: item.profile.id,
                  name: item.profile.firstName,
                  photoId: item.profile.photos[0]?.id ?? null
                })
              }
              subtitle={`${mine ? "Vous : " : ""}${item.lastMessage.body}`}
              subtitleBold={item.unreadCount > 0}
              testID={`conversation-${item.profile.firstName}`}
              title={item.profile.firstName}
              trailing={<Badge count={item.unreadCount} />}
            />
          );
        }}
        showsVerticalScrollIndicator={false}
      />
      <ActionSheet
        actions={[{ label: "Supprimer la conversation", destructive: true, onPress: () => selected && hide.mutate(selected.id) }]}
        onClose={() => setSelected(null)}
        title="Elle disparaît de votre liste jusqu'au prochain message ; l'autre personne ne la perd pas."
        visible={selected !== null}
      />
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  header: { gap: spacing.md, paddingBottom: spacing.sm },
  rail: { gap: spacing.sm },
  railContent: { gap: spacing.lg, paddingRight: spacing.lg },
  railItem: { alignItems: "center", gap: spacing.xs, width: 76 },
  list: { flexGrow: 1, paddingBottom: spacing.xxxl },
  hint: { textAlign: "center", paddingVertical: spacing.xl }
});
