import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import {
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  TextInput,
  View
} from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { ApiError } from "../api/client";
import { chatApi, discoveryApi } from "../api/endpoints";
import type { Conversation, Message } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { useSafetyDialogs } from "../features/SafetyMenu";
import { errorMessage } from "../lib/errors";
import { formatTimestamp } from "../lib/dates";
import { flattenMessages, insertMessage, type MessagePages } from "../lib/messages";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParams } from "../navigation/types";
import { radii, useTheme } from "../theme";
import { Dialog, EmptyState, ErrorState, LoadingState, PhotoImage, Text, useToast } from "../ui";

type Props = NativeStackScreenProps<MainStackParams, "Chat">;

export function ChatScreen({ route, navigation }: Props) {
  const { conversationId, peerId, peerName, peerPhotoId } = route.params;
  const t = useTheme();
  const { user } = useAuth();
  const toast = useToast();
  const qc = useQueryClient();
  const [text, setText] = useState("");
  const [menu, setMenu] = useState(false);
  const [confirm, setConfirm] = useState<"hide" | "unmatch" | null>(null);
  const listRef = useRef<FlatList<Message>>(null);

  const leave = useCallback(() => {
    void qc.invalidateQueries({ queryKey: queryKeys.conversations });
    qc.removeQueries({ queryKey: queryKeys.messages(conversationId) });
    navigation.goBack();
  }, [qc, conversationId, navigation]);
  const safety = useSafetyDialogs({ userId: peerId, name: peerName, onDone: leave });

  const q = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (last) => {
      const oldest = last.messages[last.messages.length - 1];
      return last.hasMore && oldest ? oldest.id : undefined;
    }
  });
  const messages = flattenMessages(q.data as MessagePages | undefined);
  const newestId = messages[0]?.id;
  const newestIsMine = messages[0]?.senderId === user?.id;

  // Mark as read on open and whenever a message from the peer arrives while visible.
  const markRead = useMutation({
    mutationFn: () => chatApi.markRead(conversationId),
    onSuccess: () =>
      qc.setQueryData<Conversation[]>(queryKeys.conversations, (old) =>
        old?.map((c) => (c.id === conversationId ? { ...c, unreadCount: 0 } : c))
      )
  });
  useEffect(() => {
    if (newestId !== undefined && !newestIsMine) markRead.mutate();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [newestId, newestIsMine]);

  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${peerName}`}
          onPress={() => navigation.navigate("UserProfile", { userId: peerId })}
          style={styles.title}
        >
          <PhotoImage photoId={peerPhotoId} style={styles.titleAvatar} />
          <Text variant="heading">{peerName}</Text>
        </Pressable>
      ),
      headerRight: () => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Options de la conversation"
          onPress={() => setMenu(true)}
          hitSlop={12}
        >
          <Ionicons name="ellipsis-horizontal" size={24} color={t.text} />
        </Pressable>
      )
    });
  }, [navigation, peerId, peerName, peerPhotoId, t.text]);

  const send = useMutation({
    mutationFn: (body: string) => chatApi.send(conversationId, body),
    onSuccess: (msg) => {
      qc.setQueryData<MessagePages>(queryKeys.messages(conversationId), (old) =>
        insertMessage(old, msg)
      );
      qc.setQueryData<Conversation[]>(queryKeys.conversations, (old) =>
        old
          ? [
              ...old
                .filter((c) => c.id === conversationId)
                .map((c) => ({
                  ...c,
                  lastMessage: {
                    id: msg.id,
                    senderId: msg.senderId,
                    body: msg.body,
                    createdAt: msg.createdAt
                  },
                  activityAt: msg.createdAt,
                  unreadCount: 0
                })),
              ...old.filter((c) => c.id !== conversationId)
            ]
          : old
      );
    },
    onError: (err, body) => {
      setText((cur) => cur || body);
      toast.show(errorMessage(err), "error");
    }
  });

  const submit = () => {
    const body = text.trim();
    if (!body || send.isPending) return;
    setText("");
    send.mutate(body);
  };

  const hide = useMutation({
    mutationFn: () => chatApi.hide(conversationId),
    onSuccess: leave,
    onError: (e) => toast.show(errorMessage(e), "error")
  });
  const unmatch = useMutation({
    mutationFn: async () => {
      const convo = qc
        .getQueryData<Conversation[]>(queryKeys.conversations)
        ?.find((c) => c.id === conversationId);
      if (!convo) throw new Error("missing");
      await discoveryApi.unmatch(convo.matchId);
    },
    onSuccess: () => {
      toast.show("Match annulé.");
      leave();
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  // The conversation disappears (unmatch / block by the other side): leave gracefully.
  const gone = q.error instanceof ApiError && q.error.status === 404;

  if (q.isLoading) return <LoadingState />;
  if (gone) {
    return (
      <EmptyState
        icon="heart-dislike-outline"
        title="Conversation terminée"
        message="Ce match n'est plus actif."
        actionLabel="Retour"
        onAction={leave}
      />
    );
  }
  if (q.isError)
    return <ErrorState message={errorMessage(q.error)} onRetry={() => void q.refetch()} />;

  const lastMineIndex = messages.findIndex((m) => m.senderId === user?.id);

  return (
    <SafeAreaView style={[styles.flex, { backgroundColor: t.bg }]} edges={["bottom"]}>
      <KeyboardAvoidingView
        style={styles.flex}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}
      >
        <FlatList
          ref={listRef}
          inverted
          data={messages}
          keyExtractor={(m) => String(m.id)}
          contentContainerStyle={styles.list}
          onEndReached={() => {
            if (q.hasNextPage && !q.isFetchingNextPage) void q.fetchNextPage();
          }}
          onEndReachedThreshold={0.4}
          ListFooterComponent={q.isFetchingNextPage ? <LoadingState label="" /> : null}
          ListEmptyComponent={
            <View style={styles.invertedEmpty}>
              <Text tone="muted" style={styles.centerText}>
                C'est un match ! Dites bonjour à {peerName} 👋
              </Text>
            </View>
          }
          renderItem={({ item, index }) => {
            const mine = item.senderId === user?.id;
            const showTime = index === 0 || messages[index - 1]?.senderId !== item.senderId;
            return (
              <View style={[styles.bubbleRow, mine ? styles.right : styles.left]}>
                <View
                  style={[styles.bubble, { backgroundColor: mine ? t.bubbleMine : t.bubbleTheirs }]}
                >
                  <Text style={{ color: mine ? "#fff" : t.text }}>{item.body}</Text>
                </View>
                {showTime || index === lastMineIndex ? (
                  <Text variant="caption" tone="muted" style={styles.meta}>
                    {formatTimestamp(item.createdAt)}
                    {mine && index === lastMineIndex && item.read ? " · Lu" : ""}
                  </Text>
                ) : null}
              </View>
            );
          }}
        />
        <View style={[styles.composer, { borderColor: t.border, backgroundColor: t.surface }]}>
          <TextInput
            accessibilityLabel="Votre message"
            value={text}
            onChangeText={setText}
            placeholder="Écrire un message…"
            placeholderTextColor={t.textMuted}
            multiline
            maxLength={2000}
            style={[styles.input, { color: t.text, backgroundColor: t.bg }]}
          />
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Envoyer"
            disabled={!text.trim() || send.isPending}
            onPress={submit}
            style={[styles.send, { backgroundColor: t.primary, opacity: text.trim() ? 1 : 0.4 }]}
          >
            <Ionicons name="arrow-up" size={22} color={t.onPrimary} />
          </Pressable>
        </View>
      </KeyboardAvoidingView>

      <Dialog visible={menu} title={peerName} onCancel={() => setMenu(false)}>
        <View>
          {[
            {
              icon: "person-outline",
              label: "Voir le profil",
              run: () => navigation.navigate("UserProfile", { userId: peerId })
            },
            {
              icon: "trash-outline",
              label: "Supprimer la conversation",
              run: () => setConfirm("hide")
            },
            {
              icon: "heart-dislike-outline",
              label: "Annuler le match",
              run: () => setConfirm("unmatch")
            },
            { icon: "flag-outline", label: "Signaler", run: safety.openReport },
            { icon: "ban-outline", label: "Bloquer", run: safety.openBlock }
          ].map((item) => (
            <Pressable
              key={item.label}
              accessibilityRole="button"
              onPress={() => {
                setMenu(false);
                item.run();
              }}
              style={styles.menuItem}
            >
              <Ionicons
                name={item.icon as React.ComponentProps<typeof Ionicons>["name"]}
                size={20}
                color={t.text}
              />
              <Text>{item.label}</Text>
            </Pressable>
          ))}
        </View>
      </Dialog>
      <Dialog
        visible={confirm === "hide"}
        title="Supprimer la conversation ?"
        message="Elle disparaît de votre liste uniquement. Un nouveau message la fera réapparaître."
        confirmLabel="Supprimer"
        confirmVariant="danger"
        loading={hide.isPending}
        onConfirm={() => hide.mutate()}
        onCancel={() => setConfirm(null)}
      />
      <Dialog
        visible={confirm === "unmatch"}
        title="Annuler le match ?"
        message={`Vous ne pourrez plus échanger avec ${peerName}. Cette action est définitive.`}
        confirmLabel="Annuler le match"
        confirmVariant="danger"
        loading={unmatch.isPending}
        onConfirm={() => unmatch.mutate()}
        onCancel={() => setConfirm(null)}
      />
      {safety.dialogs}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  list: { padding: 14, gap: 4 },
  title: { flexDirection: "row", alignItems: "center", gap: 10 },
  titleAvatar: { width: 34, height: 34, borderRadius: 17 },
  bubbleRow: { maxWidth: "82%", gap: 2, marginVertical: 2 },
  left: { alignSelf: "flex-start", alignItems: "flex-start" },
  right: { alignSelf: "flex-end", alignItems: "flex-end" },
  bubble: { borderRadius: radii.lg, paddingHorizontal: 14, paddingVertical: 9 },
  meta: { marginHorizontal: 6 },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: 10,
    padding: 10,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    maxHeight: 120,
    borderRadius: radii.lg,
    paddingHorizontal: 14,
    paddingVertical: 10,
    fontSize: 16
  },
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" },
  invertedEmpty: { padding: 40, transform: [{ scaleY: -1 }] },
  centerText: { textAlign: "center" },
  menuItem: { flexDirection: "row", alignItems: "center", gap: 12, paddingVertical: 12 }
});
