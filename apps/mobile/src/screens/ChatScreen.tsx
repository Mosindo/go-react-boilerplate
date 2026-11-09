import React, { useEffect, useRef, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { chatApi, matchingApi, safetyApi } from "../api/endpoints";
import type { Message } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { ReportSheet } from "../features/ReportSheet";
import { keys } from "../features/keys";
import { prependMessage } from "../features/messageCache";
import { formatTime } from "../lib/dates";
import { describeError } from "../lib/validation";
import { useNav, type Route } from "../nav/Navigation";
import { radii, useTheme } from "../theme/theme";
import { ActionSheet, ErrorState, LoadingView, PhotoImage, Screen, Text, useToast } from "../ui";

type ChatRoute = Extract<Route, { name: "chat" }>;

export default function ChatScreen({ route }: { route: ChatRoute }) {
  const { colors } = useTheme();
  const nav = useNav();
  const toast = useToast();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const { conversationId, peer, matchId } = route;
  const [draft, setDraft] = useState("");
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);
  const listRef = useRef<FlatList<Message>>(null);

  const query = useInfiniteQuery({
    queryKey: keys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => (last.length >= 30 ? last[last.length - 1].id : undefined)
  });

  const messages = query.data?.pages.flat() ?? [];
  const newestFromPeer = messages.find((m) => m.senderId !== user?.id);

  const markRead = useMutation({
    mutationFn: () => chatApi.markRead(conversationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      void queryClient.invalidateQueries({ queryKey: keys.notifications });
    }
  });

  // Mark as read on open and whenever a new message from the other person arrives while the chat is open.
  const { mutate: markConversationRead } = markRead;
  const hasUnreadFromPeer = !!newestFromPeer && !newestFromPeer.readAt;
  const newestFromPeerId = newestFromPeer?.id;
  useEffect(() => {
    if (query.isSuccess && hasUnreadFromPeer) {
      markConversationRead();
    }
  }, [query.isSuccess, hasUnreadFromPeer, newestFromPeerId, markConversationRead]);

  const send = useMutation({
    mutationFn: (body: string) => chatApi.send(conversationId, body),
    onSuccess: (message) => {
      prependMessage(queryClient, message);
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
    },
    onError: (error, body) => {
      setDraft((current) => current || body);
      toast.show(describeError(error), "error");
    }
  });

  const submit = () => {
    const body = draft.trim();
    if (!body || send.isPending) return;
    setDraft("");
    send.mutate(body);
  };

  const leave = () => {
    void queryClient.invalidateQueries({ queryKey: keys.conversations });
    nav.back();
  };

  const unmatch = useMutation({
    mutationFn: () => matchingApi.unmatch(matchId as string),
    onSuccess: leave,
    onError: (e) => toast.show(describeError(e), "error")
  });
  const block = useMutation({
    mutationFn: () => safetyApi.block(peer.userId),
    onSuccess: () => {
      toast.show("Personne bloquée.");
      leave();
    },
    onError: (e) => toast.show(describeError(e), "error")
  });
  const hide = useMutation({
    mutationFn: () => chatApi.hide(conversationId),
    onSuccess: leave,
    onError: (e) => toast.show(describeError(e), "error")
  });

  const lastOwnId = messages.find((m) => m.senderId === user?.id)?.id;

  return (
    <Screen
      onBack={nav.back}
      title={peer.firstName}
      right={
        <Pressable accessibilityRole="button" accessibilityLabel="Plus d'options" onPress={() => setMenu(true)} hitSlop={10}>
          <Text variant="heading">⋯</Text>
        </Pressable>
      }
    >
      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        <Pressable
          accessibilityRole="button"
          onPress={() => nav.push({ name: "profile", userId: peer.userId })}
          style={[styles.peerBar, { borderBottomColor: colors.border }]}
        >
          <PhotoImage url={peer.photoUrl} name={peer.firstName} circle style={styles.peerAvatar} />
          <Text variant="caption" muted>
            Voir le profil de {peer.firstName}
          </Text>
        </Pressable>
        {query.isPending ? (
          <LoadingView />
        ) : query.isError ? (
          <ErrorState message={describeError(query.error, "Conversation indisponible.")} onRetry={() => void query.refetch()} />
        ) : (
          <FlatList
            ref={listRef}
            inverted
            data={messages}
            keyExtractor={(m) => m.id}
            contentContainerStyle={styles.list}
            onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
            ListEmptyComponent={
              <Text muted style={styles.empty}>
                C'est un match ! Envoyez le premier message à {peer.firstName}.
              </Text>
            }
            renderItem={({ item }) => {
              const mine = item.senderId === user?.id;
              return (
                <View style={[styles.bubbleWrap, mine ? styles.mine : styles.theirs]}>
                  <View style={[styles.bubble, { backgroundColor: mine ? colors.primary : colors.surfaceAlt }]}>
                    <Text color={mine ? colors.onPrimary : colors.text}>{item.body}</Text>
                  </View>
                  <Text variant="caption" muted>
                    {formatTime(item.createdAt)}
                    {mine && item.id === lastOwnId ? (item.readAt ? " · Lu" : " · Envoyé") : ""}
                  </Text>
                </View>
              );
            }}
          />
        )}
        <View style={[styles.composer, { borderTopColor: colors.border, backgroundColor: colors.background }]}>
          <TextInput
            accessibilityLabel="Votre message"
            value={draft}
            onChangeText={setDraft}
            placeholder="Écrire un message…"
            placeholderTextColor={colors.textMuted}
            multiline
            maxLength={2000}
            style={[styles.input, { backgroundColor: colors.surface, borderColor: colors.border, color: colors.text }]}
            testID="chat-input"
          />
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Envoyer"
            disabled={!draft.trim() || send.isPending}
            onPress={submit}
            style={[styles.send, { backgroundColor: colors.primary, opacity: draft.trim() ? 1 : 0.4 }]}
            testID="chat-send"
          >
            <Text variant="heading" color={colors.onPrimary}>
              ➤
            </Text>
          </Pressable>
        </View>
      </KeyboardAvoidingView>
      <ActionSheet
        visible={menu}
        actions={[
          { label: "Supprimer la conversation", onPress: () => hide.mutate() },
          ...(matchId ? [{ label: "Annuler le match", destructive: true, onPress: () => unmatch.mutate() }] : []),
          { label: "Signaler", destructive: true, onPress: () => setReporting(true) },
          { label: "Bloquer", destructive: true, onPress: () => block.mutate() }
        ]}
        onClose={() => setMenu(false)}
      />
      <ReportSheet
        visible={reporting}
        userId={peer.userId}
        name={peer.firstName}
        onClose={() => setReporting(false)}
        onDone={() => {
          setReporting(false);
          leave();
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  peerBar: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  peerAvatar: { width: 28, height: 28 },
  list: { padding: 16, gap: 8, flexGrow: 1 },
  empty: { textAlign: "center", padding: 24, transform: [{ scaleY: -1 }] },
  bubbleWrap: { maxWidth: "80%", gap: 2 },
  mine: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { paddingHorizontal: 14, paddingVertical: 10, borderRadius: radii.lg },
  composer: { flexDirection: "row", alignItems: "flex-end", gap: 8, padding: 10, borderTopWidth: StyleSheet.hairlineWidth },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    borderRadius: 22,
    borderWidth: 1,
    paddingHorizontal: 16,
    paddingTop: 11,
    paddingBottom: 11,
    fontSize: 16
  },
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" }
});
