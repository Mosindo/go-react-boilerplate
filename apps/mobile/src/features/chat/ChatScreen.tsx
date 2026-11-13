import { useFocusEffect } from "@react-navigation/native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ActivityIndicator, FlatList, Pressable, StyleSheet, TextInput, View } from "react-native";
import { chatApi } from "../../api/chat";
import { discoveryApi } from "../../api/discovery";
import { safetyApi } from "../../api/safety";
import type { Conversation, Message } from "../../api/types";
import type { RootScreenProps } from "../../navigation/types";
import { appendMessage, useMessages, type MessagesData } from "../../state/queries";
import { queryKeys } from "../../state/queryClient";
import { useSession } from "../../state/session";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, ErrorState, Header, IconButton, LoadingState, Screen, Sheet, Text, useToast } from "../../ui";
import { describeError, isApiError } from "../../utils/errors";
import { formatMessageTime } from "../../utils/format";

type PendingMessage = Message & { pending?: boolean; failed?: boolean };

export default function ChatScreen({ navigation, route }: RootScreenProps<"Chat">) {
  const { userId: peerId, firstName = "", photoUrl } = route.params;
  const { colors, spacing, radii } = useTheme();
  const { userId } = useSession();
  const client = useQueryClient();
  const toast = useToast();
  const messages = useMessages(peerId);
  const [draft, setDraft] = useState("");
  const [pending, setPending] = useState<PendingMessage[]>([]);
  const [menu, setMenu] = useState(false);
  const inputRef = useRef<TextInput>(null);

  const markRead = useCallback(() => {
    void chatApi
      .markRead(peerId)
      .then(() => {
        client.setQueryData<Conversation[]>(queryKeys.conversations, (list) =>
          list?.map((c) => (c.user.userId === peerId ? { ...c, unreadCount: 0 } : c))
        );
        void client.invalidateQueries({ queryKey: queryKeys.notifications });
      })
      .catch(() => undefined);
  }, [client, peerId]);

  useFocusEffect(markRead);

  // Mark incoming messages as read while the conversation is open.
  const newest = messages.data?.pages[0]?.messages.at(-1);
  const newestIncomingId = newest && newest.senderUserId === peerId ? newest.id : null;
  const lastIncoming = useRef<string | null>(null);
  useEffect(() => {
    if (newestIncomingId && newestIncomingId !== lastIncoming.current) {
      lastIncoming.current = newestIncomingId;
      markRead();
    }
  }, [markRead, newestIncomingId]);

  const send = useMutation({
    mutationFn: (content: string) => chatApi.send(peerId, content),
    onMutate: (content) => {
      const temp: PendingMessage = {
        id: `pending-${Date.now()}`,
        senderUserId: userId ?? "",
        recipientUserId: peerId,
        content,
        createdAt: new Date().toISOString(),
        pending: true
      };
      setPending((p) => [...p, temp]);
      return temp.id;
    },
    onSuccess: (message, _content, tempId) => {
      client.setQueryData<MessagesData>(queryKeys.messages(peerId), (data) => appendMessage(data, message));
      setPending((p) => p.filter((m) => m.id !== tempId));
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
    },
    onError: (error, _content, tempId) => {
      setPending((p) => p.map((m) => (m.id === tempId ? { ...m, pending: false, failed: true } : m)));
      if (isApiError(error, 404)) {
        toast.show("Cette conversation n'est plus disponible.", "error");
        navigation.goBack();
      } else {
        toast.show(describeError(error, "Message non envoyé."), "error");
      }
    }
  });

  const submit = () => {
    const content = draft.trim();
    if (!content) return;
    setDraft("");
    send.mutate(content);
  };

  const peerLastReadAt = messages.data?.pages[0]?.peerLastReadAt ?? null;
  const items = useMemo(() => {
    const confirmed = (messages.data?.pages ?? []).flatMap((p) => p.messages);
    // Inverted list: newest first.
    return [...confirmed, ...pending].reverse() as PendingMessage[];
  }, [messages.data, pending]);
  const lastSeenOwnId = useMemo(() => {
    if (!peerLastReadAt) return null;
    const own = items.find((m) => m.senderUserId === userId && !m.pending && m.createdAt <= peerLastReadAt);
    return own?.id ?? null;
  }, [items, peerLastReadAt, userId]);

  const leave = (message: string) => {
    void client.invalidateQueries({ queryKey: queryKeys.conversations });
    client.removeQueries({ queryKey: queryKeys.messages(peerId) });
    toast.show(message, "success");
    navigation.goBack();
  };

  const actions = [
    {
      label: "Voir le profil",
      icon: "person-outline" as const,
      onPress: () => navigation.navigate("ProfileDetail", { userId: peerId, fromChat: true })
    },
    {
      label: "Effacer la conversation (pour moi)",
      icon: "trash-outline" as const,
      onPress: () =>
        void chatApi.clear(peerId).then(() => {
          client.removeQueries({ queryKey: queryKeys.messages(peerId) });
          void messages.refetch();
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
        })
    },
    {
      label: "Annuler le match",
      icon: "heart-dislike-outline" as const,
      destructive: true,
      onPress: () =>
        void discoveryApi
          .unmatch(peerId)
          .then(() => leave("Match annulé."))
          .catch((e) => toast.show(describeError(e), "error"))
    },
    {
      label: "Bloquer",
      icon: "ban-outline" as const,
      destructive: true,
      testID: "chat-block",
      onPress: () =>
        void safetyApi
          .block(peerId)
          .then(() => leave(`${firstName || "Ce membre"} est bloqué·e.`))
          .catch((e) => toast.show(describeError(e), "error"))
    },
    {
      label: "Signaler",
      icon: "flag-outline" as const,
      destructive: true,
      onPress: () => navigation.navigate("Report", { userId: peerId, firstName })
    }
  ];

  return (
    <Screen padded={false} testID="chat-screen">
      <View style={{ paddingHorizontal: spacing.md }}>
        <Header
          back
          right={
            <IconButton
              icon="ellipsis-horizontal"
              accessibilityLabel="Options de la conversation"
              onPress={() => setMenu(true)}
              testID="chat-menu"
            />
          }
        >
          <Pressable
            onPress={() => navigation.navigate("ProfileDetail", { userId: peerId, fromChat: true })}
            accessibilityRole="button"
            accessibilityLabel={`Voir le profil de ${firstName}`}
            style={styles.peer}
          >
            <Avatar url={photoUrl} name={firstName} size={34} />
            <Text variant="heading" numberOfLines={1}>
              {firstName}
            </Text>
          </Pressable>
        </Header>
      </View>

      {messages.isLoading ? (
        <LoadingState />
      ) : messages.isError ? (
        <ErrorState message={describeError(messages.error)} onAction={() => void messages.refetch()} />
      ) : (
        <FlatList
          inverted
          data={items}
          keyExtractor={(m) => m.id}
          contentContainerStyle={{ padding: spacing.md, gap: 6 }}
          onEndReached={() => {
            if (messages.hasNextPage && !messages.isFetchingNextPage) void messages.fetchNextPage();
          }}
          onEndReachedThreshold={0.3}
          ListFooterComponent={messages.isFetchingNextPage ? <ActivityIndicator color={colors.primary} /> : null}
          ListEmptyComponent={
            <View style={[styles.empty, { transform: [{ scaleY: -1 }] }]}>
              <Avatar url={photoUrl} name={firstName} size={88} />
              <Text variant="heading" align="center">
                Vous avez matché avec {firstName}
              </Text>
              <Text tone="muted" align="center">
                Lancez la conversation avec une question sur son profil.
              </Text>
            </View>
          }
          renderItem={({ item }) => {
            const mine = item.senderUserId === userId;
            return (
              <View style={[styles.bubbleWrap, { alignItems: mine ? "flex-end" : "flex-start" }]}>
                <View
                  style={[
                    styles.bubble,
                    {
                      backgroundColor: mine ? colors.primary : colors.surface,
                      borderColor: mine ? colors.primary : colors.border,
                      borderRadius: radii.lg,
                      borderBottomRightRadius: mine ? 6 : radii.lg,
                      borderBottomLeftRadius: mine ? radii.lg : 6,
                      opacity: item.pending ? 0.6 : 1
                    }
                  ]}
                  testID="chat-message"
                >
                  <Text style={{ color: mine ? colors.onPrimary : colors.text }} selectable>
                    {item.content}
                  </Text>
                </View>
                <Text variant="caption" tone={item.failed ? "danger" : "subtle"}>
                  {item.failed
                    ? "Non envoyé"
                    : item.pending
                      ? "Envoi…"
                      : `${formatMessageTime(item.createdAt)}${item.id === lastSeenOwnId ? " · Vu" : ""}`}
                </Text>
              </View>
            );
          }}
        />
      )}

      <View
        style={[
          styles.composer,
          { borderTopColor: colors.border, padding: spacing.sm, backgroundColor: colors.background }
        ]}
      >
        <TextInput
          ref={inputRef}
          value={draft}
          onChangeText={setDraft}
          placeholder="Votre message…"
          placeholderTextColor={colors.textSubtle}
          multiline
          numberOfLines={1}
          maxLength={2000}
          accessibilityLabel="Votre message"
          style={[
            styles.input,
            { color: colors.text, backgroundColor: colors.surface, borderColor: colors.border, borderRadius: radii.lg }
          ]}
          testID="chat-input"
        />
        <IconButton
          icon="arrow-up"
          filled
          tone="primary"
          accessibilityLabel="Envoyer"
          disabled={!draft.trim()}
          onPress={submit}
          testID="chat-send"
          style={{ opacity: draft.trim() ? 1 : 0.5 }}
        />
      </View>
      <Sheet visible={menu} title={firstName} onClose={() => setMenu(false)} actions={actions} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  peer: { flexDirection: "row", alignItems: "center", gap: 10 },
  empty: { alignItems: "center", gap: 10, paddingVertical: 48, paddingHorizontal: 24 },
  bubbleWrap: { gap: 2 },
  bubble: { maxWidth: "80%", paddingHorizontal: 14, paddingVertical: 10, borderWidth: StyleSheet.hairlineWidth },
  composer: { flexDirection: "row", alignItems: "flex-end", gap: 8, borderTopWidth: StyleSheet.hairlineWidth },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 140,
    paddingHorizontal: 14,
    paddingVertical: 11,
    fontSize: 16,
    borderWidth: 1
  }
});
