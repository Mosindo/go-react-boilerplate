import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { Alert, FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { blockUser, getConversations, getMessages, hideConversation, markConversationRead, sendMessage, unmatch } from "../api/dating";
import type { Message, MessagesPage } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { ReportSheet } from "../components/ReportSheet";
import { formatTime } from "../lib/format";
import { flattenMessages, prependMessage } from "../lib/messages";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParamList } from "../navigation/types";
import { SafeAreaLayout } from "../shared/layout";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Text, colors, radii, spacing } from "../shared/ui";

type Props = NativeStackScreenProps<MainStackParamList, "Chat">;

const MAX_LENGTH = 2000;

export default function ChatScreen({ navigation, route }: Props) {
  const { conversationId } = route.params;
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [text, setText] = useState("");
  const [pending, setPending] = useState<Message[]>([]);
  const [failed, setFailed] = useState<string | null>(null);
  const [reporting, setReporting] = useState(false);
  const listRef = useRef<FlatList<Message>>(null);

  const conversations = useQuery({ queryKey: queryKeys.conversations, queryFn: getConversations });
  const conversation = conversations.data?.find((c) => c.id === conversationId);

  const messages = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => getMessages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last: MessagesPage) => last.nextBefore
  });

  const items = useMemo(() => [...pending, ...flattenMessages(messages.data?.pages)], [pending, messages.data]);
  const newest = messages.data?.pages[0]?.messages[0];
  const otherLastReadAt = messages.data?.pages[0]?.otherLastReadAt;

  // Mark as read whenever a message from the other person is visible.
  const markRead = useMutation({
    mutationFn: () => markConversationRead(conversationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
    }
  });
  const markReadRef = useRef(markRead.mutate);
  markReadRef.current = markRead.mutate;
  const newestId = newest?.id;
  const newestSender = newest?.senderId;
  useEffect(() => {
    if (newestId && newestSender !== user?.id) {
      markReadRef.current();
    }
  }, [newestId, newestSender, user?.id]);

  const leave = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
    navigation.goBack();
  }, [navigation, queryClient]);

  const hide = useMutation({ mutationFn: () => hideConversation(conversationId), onSuccess: leave });
  const unmatchMutation = useMutation({
    mutationFn: async () => {
      const matchId = conversation?.matchId;
      if (matchId) await unmatch(matchId);
    },
    onSuccess: leave
  });
  const block = useMutation({
    mutationFn: async () => {
      if (conversation) await blockUser(conversation.user.id);
    },
    onSuccess: leave
  });

  const send = useMutation({
    mutationFn: (body: string) => sendMessage(conversationId, body),
    onSuccess: (message, body) => {
      setPending((list) => list.filter((m) => m.body !== body || !m.pending));
      queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conversationId), (old) =>
        old ? { ...old, pages: prependMessage(old.pages, message) } : old
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    },
    onError: (_err, body) => {
      setPending((list) => list.filter((m) => m.body !== body || !m.pending));
      setText(body);
      setFailed("Message non envoyé. Vérifiez votre connexion ou que la conversation est toujours active.");
    }
  });

  const submit = () => {
    const body = text.trim();
    if (!body || !user) return;
    setFailed(null);
    setText("");
    setPending((list) => [
      { id: `pending-${Date.now()}`, conversationId, senderId: user.id, body, createdAt: new Date().toISOString(), pending: true },
      ...list
    ]);
    send.mutate(body);
  };

  const name = conversation?.user.firstName ?? "Conversation";
  useLayoutEffect(() => {
    navigation.setOptions({
      title: name,
      headerRight: () => (
        <Pressable
          accessibilityLabel="Options de la conversation"
          hitSlop={12}
          onPress={() =>
            Alert.alert(name, undefined, [
              ...(conversation
                ? [{ text: "Voir le profil", onPress: () => navigation.navigate("ProfileDetail", { id: conversation.user.id }) }]
                : []),
              { text: "Supprimer la conversation", onPress: () => hide.mutate() },
              { text: "Annuler le match", onPress: () => unmatchMutation.mutate() },
              { text: "Signaler", onPress: () => setReporting(true) },
              { text: "Bloquer", style: "destructive", onPress: () => block.mutate() },
              { text: "Fermer", style: "cancel" }
            ])
          }
        >
          <Text tone="primary" weight="bold">
            ⋯
          </Text>
        </Pressable>
      )
    });
  }, [navigation, name, conversation, hide, unmatchMutation, block]);

  if (messages.isLoading)
    return (
      <SafeAreaLayout edges={["bottom"]}>
        <LoadingView label="Chargement…" />
      </SafeAreaLayout>
    );
  if (messages.isError) {
    return (
      <SafeAreaLayout edges={["bottom"]}>
        <ErrorView actionLabel="Retour" message="Cette conversation n'est plus disponible." onAction={leave} />
      </SafeAreaLayout>
    );
  }

  const lastOwnId = items.find((m) => m.senderId === user?.id && !m.pending)?.id;

  return (
    <SafeAreaLayout edges={["bottom"]} style={styles.root}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={90} style={styles.root}>
        <FlatList
          ListEmptyComponent={<EmptyView message={`Envoyez le premier message à ${name}.`} title="Dites bonjour 👋" />}
          contentContainerStyle={styles.list}
          data={items}
          inverted={items.length > 0}
          keyExtractor={(m) => m.id}
          onEndReached={() => messages.hasNextPage && !messages.isFetchingNextPage && void messages.fetchNextPage()}
          onEndReachedThreshold={0.4}
          ref={listRef}
          renderItem={({ item }) => {
            const mine = item.senderId === user?.id;
            const read = mine && item.id === lastOwnId && otherLastReadAt && new Date(otherLastReadAt) >= new Date(item.createdAt);
            return (
              <View style={[styles.bubbleWrap, mine ? styles.mine : styles.theirs]}>
                <View style={[styles.bubble, mine ? styles.bubbleMine : styles.bubbleTheirs, item.pending ? styles.pendingBubble : null]}>
                  <Text tone={mine ? "inverse" : "default"}>{item.body}</Text>
                </View>
                <Text tone="muted" variant="caption">
                  {item.pending ? "Envoi…" : formatTime(item.createdAt)}
                  {read ? " · Lu" : ""}
                </Text>
              </View>
            );
          }}
        />
        {failed ? (
          <Text style={styles.error} tone="danger" variant="caption">
            {failed}
          </Text>
        ) : null}
        <View style={styles.composer}>
          <TextInput
            accessibilityLabel="Votre message"
            maxLength={MAX_LENGTH}
            multiline
            onChangeText={setText}
            placeholder="Votre message"
            placeholderTextColor={colors.textSubtle}
            style={styles.input}
            testID="chat-input"
            value={text}
          />
          <Pressable
            accessibilityLabel="Envoyer"
            accessibilityRole="button"
            disabled={!text.trim()}
            onPress={submit}
            style={[styles.send, !text.trim() ? styles.sendDisabled : null]}
            testID="chat-send"
          >
            <Text tone="inverse" weight="bold">
              Envoyer
            </Text>
          </Pressable>
        </View>
      </KeyboardAvoidingView>
      <ReportSheet
        name={name}
        onClose={(reported) => {
          setReporting(false);
          if (reported) leave();
        }}
        userId={reporting && conversation ? conversation.user.id : null}
      />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.background },
  list: { paddingHorizontal: spacing.lg, paddingVertical: spacing.md, gap: spacing.sm },
  bubbleWrap: { maxWidth: "82%", gap: 2 },
  mine: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { paddingHorizontal: spacing.md, paddingVertical: spacing.sm, borderRadius: radii.lg },
  bubbleMine: { backgroundColor: colors.primary, borderBottomRightRadius: 4 },
  bubbleTheirs: { backgroundColor: colors.surface, borderWidth: 1, borderColor: colors.border, borderBottomLeftRadius: 4 },
  pendingBubble: { opacity: 0.6 },
  error: { paddingHorizontal: spacing.lg, paddingBottom: spacing.xs },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: 1,
    borderTopColor: colors.border,
    backgroundColor: colors.backgroundElevated
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    paddingHorizontal: spacing.md,
    paddingTop: 12,
    paddingBottom: 12,
    borderRadius: radii.lg,
    backgroundColor: colors.surfaceMuted,
    color: colors.text,
    fontSize: 15
  },
  send: {
    height: 44,
    paddingHorizontal: spacing.lg,
    borderRadius: radii.lg,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center"
  },
  sendDisabled: { opacity: 0.45 }
});
