import React from "react";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { chatApi } from "../../api/endpoints";
import type { Conversation } from "../../api/types";
import { keys, useConversations } from "../../hooks/queries";
import type { MainStackParams } from "../../navigation/types";
import { confirm } from "../../shared/feedback/confirm";
import { Screen } from "../../shared/layout";
import { Avatar, Badge, EmptyState, ErrorState, ListSkeleton, Text, spacing, useColors, useToast } from "../../shared/ui";
import { formatListTime } from "../../utils/time";

export function ConversationsScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const qc = useQueryClient();
  const toast = useToast();
  const c = useColors();
  const query = useConversations();
  const hide = useMutation({
    mutationFn: (id: string) => chatApi.hide(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.conversations }),
    onError: (err) => toast.show(errorMessage(err), "error"),
  });

  const items: Conversation[] = query.data?.pages.flatMap((p) => p.conversations) ?? [];

  let body: React.ReactNode;
  if (query.isPending) body = <ListSkeleton />;
  else if (query.isError) body = <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  else if (items.length === 0) {
    body = <EmptyState testID="messages-empty" title="Pas encore de conversation" message="Quand vous aurez un match, vous pourrez discuter ici. Likez quelques profils pour commencer." />;
  } else {
    body = (
      <FlatList
        data={items}
        keyExtractor={(i) => i.id}
        refreshControl={<RefreshControl refreshing={query.isRefetching && !query.isFetchingNextPage} onRefresh={() => void query.refetch()} />}
        onEndReached={() => { if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage(); }}
        onEndReachedThreshold={0.4}
        renderItem={({ item }) => (
          <Pressable
            testID={`conversation-${item.with.userId}`}
            accessibilityRole="button"
            accessibilityLabel={`Conversation avec ${item.with.firstName}${item.unread ? `, ${item.unread} non lus` : ""}. Appui long pour masquer.`}
            onPress={() => navigation.navigate("Chat", { conversationId: item.id, name: item.with.firstName, userId: item.with.userId })}
            onLongPress={() => confirm("Masquer la conversation ?", "Elle disparaîtra de votre liste mais pas de celle de l'autre personne. Elle revient si un nouveau message arrive.", "Masquer", () => hide.mutate(item.id))}
            style={[styles.row, { borderBottomColor: c.border }]}
          >
            <Avatar path={item.with.photoUrl} name={item.with.firstName} size={56} />
            <View style={{ flex: 1, gap: 2 }}>
              <Text variant="label" style={item.unread ? { fontWeight: "800" } : undefined}>{item.with.firstName}</Text>
              {item.lastMessage ? (
                <Text numberOfLines={1} tone={item.unread ? "default" : "muted"} style={item.unread ? { fontWeight: "700" } : undefined}>
                  {item.lastMessage.body}
                </Text>
              ) : (
                <Text tone="accent" variant="label">Nouveau match — dites bonjour !</Text>
              )}
            </View>
            <View style={{ alignItems: "flex-end", gap: 6 }}>
              <Text variant="caption" tone="muted">{formatListTime(item.lastMessage?.createdAt ?? item.createdAt)}</Text>
              <Badge count={item.unread} />
            </View>
          </Pressable>
        )}
      />
    );
  }

  return (
    <Screen padded={false}>
      <View style={{ padding: spacing.lg, paddingBottom: spacing.sm }}>
        <Text variant="heading">Messages</Text>
      </View>
      {body}
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.md, minHeight: 72, borderBottomWidth: StyleSheet.hairlineWidth },
});
