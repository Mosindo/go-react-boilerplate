import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, View } from "react-native";
import { fetchMessages, hideConversation, markConversationRead, sendMessage } from "../../api/chat";
import { unmatch } from "../../api/discovery";
import { blockUser } from "../../api/safety";
import type { Message } from "../../api/types";
import { keys, useConversations } from "../../hooks/queries";
import { useCurrentUser } from "../../hooks/useAuth";
import { confirmAction } from "../../lib/platform";
import type { RootStackParamList } from "../../navigation/types";
import { ErrorView, LoadingView, showError } from "../../shared/feedback";
import { SafeAreaLayout } from "../../shared/layout";
import { ActionSheet, Avatar, Input, MessageItem, Text } from "../../shared/ui";
import { MIN_TOUCH, radius, spacing, useTheme } from "../../theme";

type Props = NativeStackScreenProps<RootStackParamList, "Chat">;
type Pending = { tempId: string; body: string; createdAt: string; failed?: boolean };

export function ChatScreen({ navigation, route }: Props) {
  const { conversationId, userId, name } = route.params;
  const theme = useTheme();
  const me = useCurrentUser();
  const queryClient = useQueryClient();
  const conversations = useConversations();
  const thumb = conversations.data?.find((c) => c.id === conversationId)?.user.thumbUrl;

  const [text, setText] = useState("");
  const [pending, setPending] = useState<Pending[]>([]);
  const [menuOpen, setMenuOpen] = useState(false);
  const sending = useRef(false);

  const query = useInfiniteQuery({
    queryKey: keys.messages(conversationId),
    queryFn: ({ pageParam }) => fetchMessages(conversationId, pageParam),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (last) => last.nextBefore ?? undefined
  });

  const messages = useMemo(() => query.data?.pages.flatMap((p) => p.items) ?? [], [query.data]);
  const otherLastReadAt = query.data?.pages[0]?.otherLastReadAt ?? null;
  const newestIncomingId = messages.find((m) => m.senderId !== me.id)?.id ?? 0;

  // Mark as read on open and whenever a new incoming message shows up while the screen is mounted.
  useEffect(() => {
    if (query.isSuccess) {
      markConversationRead(conversationId)
        .then(() => {
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
        })
        .catch(() => undefined);
    }
  }, [conversationId, query.isSuccess, newestIncomingId, queryClient]);

  const goToProfile = useCallback(() => navigation.navigate("ProfileDetail", { userId }), [navigation, userId]);

  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <Pressable
          testID="chat-header"
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${name}`}
          onPress={goToProfile}
          style={styles.headerTitle}
        >
          <Avatar name={name} uri={thumb} size={32} />
          <Text variant="label" style={{ fontSize: 17 }}>
            {name}
          </Text>
        </Pressable>
      ),
      headerRight: () => (
        <Pressable
          testID="chat-menu"
          accessibilityRole="button"
          accessibilityLabel="Plus d'options"
          onPress={() => setMenuOpen(true)}
          style={styles.menuButton}
        >
          <Ionicons name="ellipsis-horizontal" size={24} color={theme.text} />
        </Pressable>
      )
    });
  }, [navigation, name, thumb, goToProfile, theme.text]);

  const push = (msg: Message) =>
    queryClient.setQueryData(keys.messages(conversationId), (old: typeof query.data) => {
      if (!old || old.pages.length === 0) return old;
      const [first, ...rest] = old.pages;
      if (!first || old.pages.some((p) => p.items.some((m) => m.id === msg.id))) return old;
      return { ...old, pages: [{ ...first, items: [msg, ...first.items] }, ...rest] };
    });

  const send = async (draft: Pending) => {
    try {
      const msg = await sendMessage(conversationId, draft.body);
      push(msg);
      setPending((list) => list.filter((p) => p.tempId !== draft.tempId));
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
    } catch (err) {
      setPending((list) => list.map((p) => (p.tempId === draft.tempId ? { ...p, failed: true } : p)));
      showError(err instanceof Error ? err.message : "Message non envoyé.");
    }
  };

  const submit = () => {
    const body = text.trim();
    if (!body || sending.current) return;
    const draft: Pending = { tempId: `${Date.now()}-${Math.random()}`, body, createdAt: new Date().toISOString() };
    setText("");
    setPending((list) => [draft, ...list]);
    sending.current = true;
    void send(draft).finally(() => {
      sending.current = false;
    });
  };

  const leave = async (action: () => Promise<unknown>, done: string) => {
    try {
      await action();
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      void queryClient.invalidateQueries({ queryKey: keys.notifications });
      navigation.goBack();
    } catch (err) {
      showError(err instanceof Error ? err.message : done);
    }
  };

  const items = [...pending.map((p) => ({ kind: "pending" as const, p })), ...messages.map((m) => ({ kind: "sent" as const, m }))];

  return (
    <SafeAreaLayout edges={["bottom", "left", "right"]}>
      <KeyboardAvoidingView
        style={styles.flex}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}
      >
        <View style={[styles.flex, styles.column]}>
          {query.isPending ? (
            <LoadingView />
          ) : query.isError ? (
            <ErrorView message="Conversation introuvable." onRetry={() => void query.refetch()} />
          ) : (
            <FlatList
              testID="chat-list"
              inverted
              data={items}
              keyExtractor={(it) => (it.kind === "pending" ? it.p.tempId : String(it.m.id))}
              onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
              onEndReachedThreshold={0.4}
              contentContainerStyle={{ paddingVertical: spacing.md }}
              ListEmptyComponent={
                <View style={styles.emptyChat}>
                  <Text tone="muted" center>
                    C'est le début de votre conversation avec {name}. Lancez-vous !
                  </Text>
                </View>
              }
              renderItem={({ item }) =>
                item.kind === "pending" ? (
                  <Pressable
                    disabled={!item.p.failed}
                    onPress={() => {
                      setPending((l) => l.filter((x) => x.tempId !== item.p.tempId));
                      void send({ ...item.p, failed: false });
                    }}
                  >
                    <MessageItem body={item.p.body} createdAt={item.p.createdAt} mine pending={!item.p.failed} />
                    {item.p.failed ? (
                      <Text variant="caption" tone="danger" style={styles.failed}>
                        Échec — touchez pour réessayer
                      </Text>
                    ) : null}
                  </Pressable>
                ) : (
                  <MessageItem
                    body={item.m.body}
                    createdAt={item.m.createdAt}
                    mine={item.m.senderId === me.id}
                    read={item.m.senderId === me.id && otherLastReadAt !== null && new Date(otherLastReadAt) >= new Date(item.m.createdAt)}
                  />
                )
              }
            />
          )}

          <View style={[styles.composer, { borderTopColor: theme.border, backgroundColor: theme.background }]}>
            <Input
              testID="chat-input"
              style={styles.input}
              value={text}
              onChangeText={setText}
              placeholder="Votre message…"
              maxLength={2000}
              multiline
              onKeyPress={(e) => {
                // Web: Enter sends, Shift+Enter inserts a new line.
                const native = e.nativeEvent as unknown as { key: string; shiftKey?: boolean };
                if (Platform.OS === "web" && native.key === "Enter" && !native.shiftKey) {
                  e.preventDefault?.();
                  submit();
                }
              }}
            />
            <Pressable
              testID="chat-send"
              accessibilityRole="button"
              accessibilityLabel="Envoyer"
              disabled={!text.trim()}
              onPress={submit}
              style={[styles.send, { backgroundColor: theme.primary, opacity: text.trim() ? 1 : 0.4 }]}
            >
              <Ionicons name="send" size={20} color={theme.primaryText} />
            </Pressable>
          </View>
        </View>
      </KeyboardAvoidingView>

      <ActionSheet
        visible={menuOpen}
        title={name}
        onClose={() => setMenuOpen(false)}
        actions={[
          { label: "Voir le profil", onPress: goToProfile },
          {
            label: "Supprimer la conversation (pour moi)",
            testID: "chat-hide",
            onPress: () =>
              void (async () => {
                if (
                  await confirmAction(
                    "Supprimer cette conversation ?",
                    "Elle disparaît de votre liste mais pas de celle de votre match. Elle revient si un nouveau message arrive.",
                    "Supprimer"
                  )
                ) {
                  await leave(() => hideConversation(conversationId), "Suppression impossible.");
                }
              })()
          },
          {
            label: "Annuler le match",
            onPress: () =>
              void (async () => {
                const matchId = conversations.data?.find((c) => c.id === conversationId)?.matchId;
                if (
                  matchId &&
                  (await confirmAction(
                    `Annuler le match avec ${name} ?`,
                    "La conversation sera supprimée pour vous deux.",
                    "Annuler le match"
                  ))
                ) {
                  await leave(() => unmatch(matchId), "Action impossible.");
                }
              })()
          },
          { label: "Signaler", onPress: () => navigation.navigate("Report", { userId, name }) },
          {
            label: "Bloquer",
            destructive: true,
            testID: "chat-block",
            onPress: () =>
              void (async () => {
                if (await confirmAction(`Bloquer ${name} ?`, "Vous ne vous verrez plus et la conversation sera supprimée.", "Bloquer")) {
                  await leave(() => blockUser(userId), "Blocage impossible.");
                }
              })()
          }
        ]}
      />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  column: { width: "100%", maxWidth: 680, alignSelf: "center" },
  headerTitle: { flexDirection: "row", alignItems: "center", gap: spacing.sm, minHeight: MIN_TOUCH },
  menuButton: { width: MIN_TOUCH, height: MIN_TOUCH, alignItems: "center", justifyContent: "center" },
  emptyChat: { padding: spacing.xl, transform: [{ scaleY: -1 }] },
  failed: { alignSelf: "flex-end", paddingHorizontal: spacing.lg },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: { flex: 1, minHeight: MIN_TOUCH, maxHeight: 120, borderRadius: radius.lg },
  send: { width: MIN_TOUCH, height: MIN_TOUCH, borderRadius: radius.pill, alignItems: "center", justifyContent: "center" }
});
