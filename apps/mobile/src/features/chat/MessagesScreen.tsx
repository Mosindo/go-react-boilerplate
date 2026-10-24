import React from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { chatApi } from "../../api/endpoints";
import type { Conversation } from "../../api/types";
import { formatShortTime } from "../../lib/date";
import type { MainStackParamList } from "../../navigation/types";
import { invalidateChat, queryKeys } from "../../realtime/cache";
import { radius, spacing, useTheme } from "../../theme/theme";
import {
  Avatar,
  EmptyState,
  ErrorState,
  Icon,
  LoadingState,
  Screen,
  Text,
  useDialog,
  useToast,
} from "../../ui";

export function MessagesScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const queryClient = useQueryClient();
  const dialog = useDialog();
  const toast = useToast();
  const { data, error, isLoading, isRefetching, refetch } = useQuery({
    queryKey: queryKeys.conversations,
    queryFn: () => chatApi.conversations(),
  });

  const hide = useMutation({
    mutationFn: chatApi.hide,
    onSuccess: () => invalidateChat(queryClient),
    onError: () =>
      toast.show({ message: "Impossible de supprimer la conversation.", kind: "error" }),
  });

  const open = (c: Conversation) =>
    navigation.navigate("Chat", {
      conversationId: c.id,
      userId: c.user.userId,
      name: c.user.firstName,
      photoId: c.user.photos[0]?.id,
    });

  const askHide = async (c: Conversation) => {
    const ok = await dialog.confirm({
      title: "Supprimer cette conversation ?",
      message: `Elle disparaît de votre liste, ainsi que les messages échangés. ${c.user.firstName} la conserve et le match n'est pas supprimé.`,
      confirmLabel: "Supprimer",
      destructive: true,
    });
    if (ok) hide.mutate(c.id);
  };

  if (isLoading) return <LoadingState />;
  if (error) return <ErrorState error={error} onRetry={() => void refetch()} />;

  const all = data ?? [];
  const fresh = all.filter((c) => !c.lastMessage);
  const talks = all.filter((c) => c.lastMessage);

  return (
    <Screen testID="messages-screen" edges={["top"]} padded={false}>
      <Text variant="title" style={styles.title}>
        Messages
      </Text>
      {all.length === 0 ? (
        <EmptyState
          testID="messages-empty"
          icon="chatbubbles-outline"
          title="Pas encore de match"
          message="Dès que vous vous plaisez mutuellement avec quelqu'un, la conversation apparaît ici."
          actionLabel="Découvrir des profils"
          onAction={() => navigation.navigate("Tabs", { screen: "Discover" })}
        />
      ) : (
        <FlatList
          testID="conversation-list"
          data={talks}
          keyExtractor={(c) => c.id}
          refreshControl={
            <RefreshControl refreshing={isRefetching} onRefresh={() => void refetch()} />
          }
          ListHeaderComponent={
            fresh.length > 0 ? (
              <View style={styles.section}>
                <Text variant="label" muted style={styles.sectionTitle}>
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
                      testID={`new-match-${c.user.firstName}`}
                      accessibilityRole="button"
                      accessibilityLabel={`Nouveau match avec ${c.user.firstName}`}
                      onPress={() => open(c)}
                      onLongPress={() => void askHide(c)}
                      style={styles.freshItem}
                    >
                      <Avatar photoId={c.user.photos[0]?.id} name={c.user.firstName} size={64} />
                      <Text variant="caption" numberOfLines={1}>
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
              testID={`conversation-${item.user.firstName}`}
              accessibilityRole="button"
              accessibilityLabel={`Conversation avec ${item.user.firstName}${item.unreadCount > 0 ? `, ${item.unreadCount} non lus` : ""}`}
              onPress={() => open(item)}
              onLongPress={() => void askHide(item)}
              style={[styles.row, { borderBottomColor: colors.border }]}
            >
              <Avatar photoId={item.user.photos[0]?.id} name={item.user.firstName} />
              <View style={styles.rowBody}>
                <View style={styles.rowTop}>
                  <Text variant="heading">{item.user.firstName}</Text>
                  <Text variant="caption" muted>
                    {item.lastMessage ? formatShortTime(item.lastMessage.createdAt) : ""}
                  </Text>
                </View>
                <Text
                  numberOfLines={1}
                  muted={item.unreadCount === 0}
                  style={item.unreadCount > 0 ? styles.unreadText : undefined}
                >
                  {item.lastMessage?.content}
                </Text>
              </View>
              {item.unreadCount > 0 ? (
                <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                  <Text variant="caption" color={colors.onPrimary} testID="unread-badge">
                    {item.unreadCount > 99 ? "99+" : item.unreadCount}
                  </Text>
                </View>
              ) : null}
              <Pressable
                testID={`hide-${item.user.firstName}`}
                accessibilityRole="button"
                accessibilityLabel={`Supprimer la conversation avec ${item.user.firstName}`}
                hitSlop={8}
                onPress={() => void askHide(item)}
              >
                <Icon name="trash-outline" size={20} color={colors.textMuted} />
              </Pressable>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  title: { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  section: { paddingBottom: spacing.md },
  sectionTitle: { paddingHorizontal: spacing.lg, paddingBottom: spacing.sm },
  fresh: { paddingHorizontal: spacing.lg, gap: spacing.lg },
  freshItem: { alignItems: "center", gap: spacing.xs, width: 72 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  rowBody: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  unreadText: { fontWeight: "700" },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: radius.pill,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6,
  },
});
