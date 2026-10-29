import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import React, { useMemo, useState } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { fetchConversations, hideConversation } from "../../api/chat";
import type { Conversation } from "../../api/types";
import { Avatar } from "../../components/Avatar";
import { Button } from "../../components/Button";
import { Sheet } from "../../components/Sheet";
import { EmptyView, ErrorView, LoadingView } from "../../components/StateViews";
import { queryKeys } from "../../hooks/queries";
import { errorMessage } from "../../hooks/useAsyncAction";
import { formatRelative } from "../../lib/format";
import type { MainTabParamList, RootStackParamList } from "../../navigation/types";
import { useMe } from "../../state/AuthProvider";
import { useToast } from "../../state/ToastProvider";
import { fontSizes, radii, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

type Props = CompositeScreenProps<
  BottomTabScreenProps<MainTabParamList, "Chats">,
  NativeStackScreenProps<RootStackParamList>
>;

export function useConversations() {
  return useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => fetchConversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextBefore ?? undefined,
    refetchOnMount: "always"
  });
}

export default function ConversationsScreen({ navigation }: Props) {
  const theme = useTheme();
  const toast = useToast();
  const me = useMe();
  const queryClient = useQueryClient();
  const query = useConversations();
  const [menuFor, setMenuFor] = useState<Conversation | null>(null);

  const all = useMemo(() => query.data?.pages.flatMap((p) => p.conversations) ?? [], [query.data]);
  const fresh = all.filter((c) => c.lastMessage === null);
  const active = all.filter((c) => c.lastMessage !== null);

  const open = (c: Conversation) =>
    navigation.navigate("Chat", { conversationId: c.id, user: c.user });

  const hide = async (c: Conversation) => {
    setMenuFor(null);
    try {
      await hideConversation(c.id);
      queryClient.removeQueries({ queryKey: queryKeys.messages(c.id) });
      await queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  const header = (
    <View>
      <Text accessibilityRole="header" style={[styles.title, { color: theme.text }]}>
        Messages
      </Text>
      {fresh.length > 0 ? (
        <View style={styles.freshWrap}>
          <Text style={[styles.section, { color: theme.textMuted }]}>Nouveaux matchs</Text>
          <FlatList
            data={fresh}
            horizontal
            keyExtractor={(c) => c.id}
            renderItem={({ item }) => (
              <Pressable
                accessibilityLabel={`Nouveau match : ${item.user.firstName}`}
                accessibilityRole="button"
                onPress={() => open(item)}
                style={styles.fresh}
              >
                <View style={[styles.ring, { borderColor: theme.primary }]}>
                  <Avatar name={item.user.firstName} size={64} url={item.user.photoUrl} />
                </View>
                <Text numberOfLines={1} style={[styles.freshName, { color: theme.text }]}>
                  {item.user.firstName}
                </Text>
              </Pressable>
            )}
            showsHorizontalScrollIndicator={false}
          />
          {active.length > 0 ? (
            <Text style={[styles.section, styles.sectionSpaced, { color: theme.textMuted }]}>
              Conversations
            </Text>
          ) : null}
        </View>
      ) : null}
    </View>
  );

  return (
    <SafeAreaView edges={["top"]} style={[styles.flex, { backgroundColor: theme.background }]}>
      {query.isLoading ? (
        <LoadingView />
      ) : query.isError ? (
        <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : all.length === 0 ? (
        <View style={styles.flex}>
          <Text accessibilityRole="header" style={[styles.title, { color: theme.text }]}>
            Messages
          </Text>
          <EmptyView
            actionLabel="Découvrir des profils"
            icon="chatbubbles-outline"
            message="Quand vous et une autre personne vous aimerez mutuellement, votre conversation apparaîtra ici."
            onAction={() => navigation.navigate("Discover")}
            title="Pas encore de match"
          />
        </View>
      ) : (
        <FlatList
          ListHeaderComponent={header}
          data={active}
          keyExtractor={(c) => c.id}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          refreshControl={
            <RefreshControl
              onRefresh={() => void query.refetch()}
              refreshing={query.isRefetching && !query.isFetchingNextPage}
              tintColor={theme.primary}
            />
          }
          renderItem={({ item }) => {
            const last = item.lastMessage;
            const mine = last?.senderId === me.id;
            return (
              <Pressable
                accessibilityHint="Appuyez longuement pour supprimer la conversation"
                accessibilityLabel={`${item.user.firstName}${item.unreadCount ? `, ${item.unreadCount} non lus` : ""}`}
                accessibilityRole="button"
                onLongPress={() => setMenuFor(item)}
                onPress={() => open(item)}
                style={({ pressed }) => [
                  styles.row,
                  { backgroundColor: pressed ? theme.surfaceMuted : "transparent" }
                ]}
              >
                <Avatar name={item.user.firstName} size={56} url={item.user.photoUrl} />
                <View style={styles.rowText}>
                  <View style={styles.rowTop}>
                    <Text numberOfLines={1} style={[styles.name, { color: theme.text }]}>
                      {item.user.firstName}
                    </Text>
                    {last ? (
                      <Text style={[styles.time, { color: theme.textMuted }]}>
                        {formatRelative(last.createdAt)}
                      </Text>
                    ) : null}
                  </View>
                  <View style={styles.rowTop}>
                    <Text
                      numberOfLines={1}
                      style={[
                        styles.preview,
                        { color: item.unreadCount ? theme.text : theme.textMuted },
                        item.unreadCount ? styles.bold : null
                      ]}
                    >
                      {mine ? "Vous : " : ""}
                      {last?.body}
                    </Text>
                    {item.unreadCount > 0 ? (
                      <View style={[styles.badge, { backgroundColor: theme.primary }]}>
                        <Text style={styles.badgeText}>
                          {item.unreadCount > 99 ? "99+" : item.unreadCount}
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

      <Sheet
        onClose={() => setMenuFor(null)}
        title={menuFor?.user.firstName ?? ""}
        visible={menuFor !== null}
      >
        <Text style={[styles.sheetHint, { color: theme.textMuted }]}>
          La conversation disparaît de votre côté uniquement. Elle réapparaîtra si cette personne
          vous écrit.
        </Text>
        {menuFor ? (
          <Button
            onPress={() => void hide(menuFor)}
            title="Supprimer la conversation"
            variant="danger"
          />
        ) : null}
      </Sheet>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  title: {
    fontSize: 30,
    fontWeight: "700",
    letterSpacing: -0.5,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  },
  section: {
    fontSize: fontSizes.xs,
    fontWeight: "700",
    letterSpacing: 0.8,
    textTransform: "uppercase",
    paddingHorizontal: spacing.lg,
    marginBottom: spacing.sm
  },
  sectionSpaced: { marginTop: spacing.lg },
  freshWrap: { paddingBottom: spacing.sm },
  fresh: { alignItems: "center", width: 80, marginLeft: spacing.sm },
  ring: { borderWidth: 2, borderRadius: 40, padding: 3 },
  freshName: { fontSize: fontSizes.sm, fontWeight: "600", marginTop: spacing.xs, maxWidth: 76 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  },
  rowText: { flex: 1, marginLeft: spacing.md },
  rowTop: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  name: { fontSize: fontSizes.md, fontWeight: "700", flex: 1 },
  time: { fontSize: fontSizes.xs, marginLeft: spacing.sm },
  preview: { fontSize: fontSizes.sm, flex: 1, marginTop: 2 },
  bold: { fontWeight: "700" },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: radii.pill,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6,
    marginLeft: spacing.sm
  },
  badgeText: { color: "#FFFFFF", fontSize: 12, fontWeight: "700" },
  sheetHint: { fontSize: fontSizes.sm, lineHeight: 20, marginBottom: spacing.lg }
});
