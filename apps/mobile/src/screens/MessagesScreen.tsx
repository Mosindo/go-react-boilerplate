import React, { useState } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { chatApi } from "../api/endpoints";
import type { Conversation } from "../api/types";
import { errorMessage, keys, useConversations } from "../hooks/queries";
import { timeLabel } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import {
  Avatar,
  EmptyView,
  ErrorView,
  LoadingView,
  Screen,
  Sheet,
  SheetAction,
  Text,
  spacing,
  useTheme,
  useToast
} from "../ui";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export default function MessagesScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<Nav>();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useConversations();
  const [selected, setSelected] = useState<Conversation | null>(null);

  const open = (c: Conversation) =>
    navigation.navigate("Chat", { conversationId: c.id, userId: c.user.userId, name: c.user.displayName });

  const hide = async (c: Conversation) => {
    setSelected(null);
    try {
      await chatApi.hide(c.id);
      await queryClient.invalidateQueries({ queryKey: keys.conversations });
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  if (query.isLoading) return <LoadingView />;
  if (query.isError) return <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;

  const all = query.data ?? [];
  const fresh = all.filter((c) => !c.lastMessage);
  const talking = all.filter((c) => c.lastMessage);

  return (
    <Screen topInset testID="messages-screen">
      <Text style={styles.title} variant="title">
        Messages
      </Text>
      {all.length === 0 ? (
        <EmptyView
          icon="chatbubbles-outline"
          message="Likez des profils : dès qu'un intérêt est réciproque, la conversation s'ouvre ici."
          title="Pas encore de match"
          testID="messages-empty"
        />
      ) : (
        <FlatList
          data={talking}
          keyExtractor={(c) => c.id}
          ListHeaderComponent={
            fresh.length > 0 ? (
              <View style={styles.section}>
                <Text tone="muted" variant="label">
                  Nouveaux matchs
                </Text>
                <ScrollView contentContainerStyle={styles.fresh} horizontal showsHorizontalScrollIndicator={false}>
                  {fresh.map((c) => (
                    <Pressable
                      accessibilityLabel={`Nouveau match avec ${c.user.displayName}`}
                      accessibilityRole="button"
                      key={c.id}
                      onPress={() => open(c)}
                      style={styles.freshItem}
                      testID="new-match"
                    >
                      <Avatar name={c.user.displayName} size={64} uri={c.user.photoUrl} />
                      <Text numberOfLines={1} variant="caption">
                        {c.user.displayName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
              </View>
            ) : null
          }
          ListEmptyComponent={
            fresh.length > 0 ? (
              <Text tone="muted">Envoyez le premier message à l'un de vos nouveaux matchs.</Text>
            ) : null
          }
          refreshControl={
            <RefreshControl
              onRefresh={() => void query.refetch()}
              refreshing={query.isRefetching}
              tintColor={colors.primary}
            />
          }
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              onLongPress={() => setSelected(item)}
              onPress={() => open(item)}
              style={styles.row}
              testID="conversation-row"
            >
              <Avatar name={item.user.displayName} uri={item.user.photoUrl} />
              <View style={styles.rowText}>
                <View style={styles.rowTop}>
                  <Text numberOfLines={1} style={styles.flex} variant="label">
                    {item.user.displayName}
                  </Text>
                  <Text tone="muted" variant="caption">
                    {timeLabel(item.lastActivityAt)}
                  </Text>
                </View>
                <View style={styles.rowTop}>
                  <Text
                    numberOfLines={1}
                    style={[styles.flex, item.unreadCount > 0 ? { fontWeight: "700" } : null]}
                    tone={item.unreadCount > 0 ? "default" : "muted"}
                  >
                    {item.lastMessage?.senderId === item.user.userId ? "" : "Vous : "}
                    {item.lastMessage?.body}
                  </Text>
                  {item.unreadCount > 0 ? (
                    <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                      <Text style={{ color: colors.primaryText, fontSize: 12, fontWeight: "700" }}>
                        {item.unreadCount}
                      </Text>
                    </View>
                  ) : null}
                </View>
              </View>
            </Pressable>
          )}
        />
      )}

      <Sheet onClose={() => setSelected(null)} title={selected?.user.displayName} visible={Boolean(selected)}>
        <SheetAction label="Masquer la conversation" onPress={() => selected && void hide(selected)} />
      </Sheet>
    </Screen>
  );
}

const styles = StyleSheet.create({
  title: { marginBottom: spacing.md },
  flex: { flex: 1 },
  section: { gap: spacing.sm, marginBottom: spacing.lg },
  fresh: { gap: spacing.lg, paddingVertical: spacing.xs },
  freshItem: { alignItems: "center", gap: spacing.xs, width: 72 },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingVertical: spacing.md, minHeight: 72 },
  rowText: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: 11,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6
  }
});
