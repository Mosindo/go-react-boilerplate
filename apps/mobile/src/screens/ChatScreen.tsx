import React, { useCallback, useEffect, useRef, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useFocusEffect } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  MESSAGES_PAGE_SIZE,
  errorMessage,
  hideConversation,
  listMessages,
  markConversationRead,
  queryKeys,
  sendMessage,
  unmatch,
  type ChatMessage
} from "../api/platform";
import { useSafetyMenu } from "../components/useSafetyMenu";
import { activeChat } from "../hooks/useRealtime";
import { useAuth } from "../hooks/useAuth";
import { formatClock } from "../lib/dates";
import { appendOlder, mergeMessage } from "../lib/messages";
import type { RootStackParamList } from "../navigation/types";
import { SafeAreaLayout } from "../shared/layout";
import { EmptyView, ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Text, colors, radii, spacing } from "../shared/ui";
import { showDialog } from "../shared/dialog";

type Props = NativeStackScreenProps<RootStackParamList, "Chat">;

const MAX_LENGTH = 2000;

function Bubble({ message, mine }: { message: ChatMessage; mine: boolean }) {
  return (
    <View style={[styles.bubbleRow, mine ? styles.mineRow : null]}>
      <View style={[styles.bubble, mine ? styles.mine : styles.theirs]}>
        <Text tone={mine ? "inverse" : "default"}>{message.body}</Text>
        <Text style={styles.meta} tone={mine ? "inverse" : "muted"} variant="caption">
          {formatClock(message.createdAt)}
          {mine && message.readAt ? " · Read" : ""}
        </Text>
      </View>
    </View>
  );
}

export default function ChatScreen({ navigation, route }: Props) {
  const { matchId, name, userId } = route.params;
  const { user } = useAuth();
  const client = useQueryClient();
  const [text, setText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(true);
  const [loadingOlder, setLoadingOlder] = useState(false);
  const loadingOlderRef = useRef(false);

  const messagesQuery = useQuery({
    queryKey: queryKeys.messages(matchId),
    queryFn: async () => {
      const page = await listMessages(matchId);
      setHasMore(page.length === MESSAGES_PAGE_SIZE);
      return page;
    },
    staleTime: 0
  });
  const messages = messagesQuery.data ?? [];

  const leave = useCallback(() => {
    void client.invalidateQueries({ queryKey: queryKeys.conversations });
    navigation.goBack();
  }, [client, navigation]);
  const safety = useSafetyMenu({ name, onBlocked: leave, userId });

  const send = useMutation({
    mutationFn: (body: string) => sendMessage(matchId, body),
    onSuccess: (message) => {
      setText("");
      client.setQueryData<ChatMessage[]>(queryKeys.messages(matchId), (old) => mergeMessage(old ?? [], message));
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
    },
    onError: (e) => setError(errorMessage(e))
  });

  const unreadIncoming = messages.filter((m) => m.senderId !== user?.id && !m.readAt).length;
  useFocusEffect(
    useCallback(() => {
      activeChat.matchId = matchId;
      return () => {
        if (activeChat.matchId === matchId) {
          activeChat.matchId = null;
        }
      };
    }, [matchId])
  );
  useEffect(() => {
    if (unreadIncoming === 0) {
      return;
    }
    markConversationRead(matchId)
      .then(() => {
        client.setQueryData<ChatMessage[]>(queryKeys.messages(matchId), (old) =>
          old?.map((m) => (m.senderId !== user?.id && !m.readAt ? { ...m, readAt: new Date().toISOString() } : m))
        );
        return client.invalidateQueries({ queryKey: queryKeys.conversations });
      })
      .catch(() => undefined);
  }, [client, matchId, unreadIncoming, user?.id]);

  // A conversation that vanished (unmatched or blocked by the other person) closes the screen.
  const gone = (messagesQuery.error as { status?: number } | null)?.status === 404;
  useEffect(() => {
    if (gone) {
      showToast("This conversation is no longer available.", { tone: "info" });
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
      navigation.goBack();
    }
  }, [client, gone, navigation]);

  const menu = useCallback(() => {
    showDialog(name, undefined, [
      { text: "View profile", onPress: () => navigation.navigate("ProfileDetail", { userId, name }) },
      {
        text: "Delete conversation",
        onPress: () =>
          showDialog("Delete this conversation?", "It disappears from your list. They keep their copy, and it returns if they write again.", [
            { text: "Cancel", style: "cancel" },
            {
              text: "Delete",
              style: "destructive",
              onPress: () =>
                hideConversation(matchId)
                  .then(leave)
                  .catch((e) => showToast(errorMessage(e), { tone: "error" }))
            }
          ])
      },
      {
        text: "Unmatch",
        style: "destructive",
        onPress: () =>
          showDialog(`Unmatch ${name}?`, "The match and all messages are deleted for both of you.", [
            { text: "Cancel", style: "cancel" },
            {
              text: "Unmatch",
              style: "destructive",
              onPress: () =>
                unmatch(matchId)
                  .then(leave)
                  .catch((e) => showToast(errorMessage(e), { tone: "error" }))
            }
          ])
      },
      { text: "Report or block…", onPress: safety.open },
      { text: "Cancel", style: "cancel" }
    ]);
  }, [leave, matchId, name, navigation, safety.open, userId]);

  useEffect(() => {
    navigation.setOptions({
      headerRight: () => (
        <Pressable accessibilityLabel="Conversation options" accessibilityRole="button" hitSlop={12} onPress={menu} testID="chat-menu">
          <Text variant="heading" weight="bold">
            ⋯
          </Text>
        </Pressable>
      )
    });
  }, [menu, navigation]);

  const loadOlder = async () => {
    const oldest = messages[messages.length - 1];
    if (!hasMore || loadingOlderRef.current || !oldest) {
      return;
    }
    loadingOlderRef.current = true;
    setLoadingOlder(true);
    try {
      const page = await listMessages(matchId, oldest.createdAt);
      setHasMore(page.length === MESSAGES_PAGE_SIZE);
      client.setQueryData<ChatMessage[]>(queryKeys.messages(matchId), (old) => appendOlder(old ?? [], page));
    } catch (e) {
      showToast(errorMessage(e), { tone: "error" });
    } finally {
      loadingOlderRef.current = false;
      setLoadingOlder(false);
    }
  };

  const submit = () => {
    const body = text.trim();
    if (!body || send.isPending) {
      return;
    }
    setError(null);
    send.mutate(body);
  };

  if (messagesQuery.isLoading) {
    return (
      <SafeAreaLayout edges={["bottom"]}>
        <LoadingView label="Opening conversation…" />
      </SafeAreaLayout>
    );
  }
  if (messagesQuery.isError && !gone) {
    return (
      <SafeAreaLayout edges={["bottom"]}>
        <ErrorView message={errorMessage(messagesQuery.error)} onAction={() => void messagesQuery.refetch()} />
      </SafeAreaLayout>
    );
  }

  return (
    <SafeAreaLayout edges={["bottom", "left", "right"]}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0} style={styles.flex}>
        <FlatList
          ListEmptyComponent={<EmptyView message={`Say hello to ${name} 👋`} style={styles.invertedFix} title="You matched!" />}
          ListFooterComponent={loadingOlder ? <LoadingView label="Loading earlier messages…" /> : null}
          contentContainerStyle={styles.list}
          data={messages}
          inverted
          keyExtractor={(m) => m.id}
          onEndReached={() => void loadOlder()}
          onEndReachedThreshold={0.4}
          renderItem={({ item }) => <Bubble message={item} mine={item.senderId === user?.id} />}
          testID="chat-list"
        />
        {error ? (
          <Text style={styles.error} tone="danger" variant="caption">
            {error}
          </Text>
        ) : null}
        <View style={styles.composer}>
          <TextInput
            accessibilityLabel="Message"
            maxLength={MAX_LENGTH}
            multiline
            onChangeText={setText}
            placeholder="Write a message"
            placeholderTextColor={colors.textSubtle}
            style={styles.input}
            testID="chat-input"
            value={text}
          />
          <Pressable
            accessibilityLabel="Send message"
            accessibilityRole="button"
            accessibilityState={{ disabled: !text.trim() || send.isPending }}
            disabled={!text.trim() || send.isPending}
            onPress={submit}
            style={[styles.send, !text.trim() || send.isPending ? styles.sendDisabled : null]}
            testID="chat-send"
          >
            <Text tone="inverse" weight="bold">
              Send
            </Text>
          </Pressable>
        </View>
      </KeyboardAvoidingView>
      {safety.element}
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  list: { padding: spacing.lg, gap: spacing.sm },
  invertedFix: { transform: [{ scaleY: -1 }] },
  bubbleRow: { flexDirection: "row" },
  mineRow: { justifyContent: "flex-end" },
  bubble: { maxWidth: "80%", paddingHorizontal: spacing.md, paddingVertical: spacing.sm, borderRadius: radii.lg, gap: 2 },
  mine: { backgroundColor: colors.primary, borderBottomRightRadius: radii.xs },
  theirs: { backgroundColor: colors.backgroundElevated, borderWidth: 1, borderColor: colors.border, borderBottomLeftRadius: radii.xs },
  meta: { alignSelf: "flex-end", opacity: 0.8 },
  error: { paddingHorizontal: spacing.lg },
  composer: { flexDirection: "row", alignItems: "flex-end", gap: spacing.sm, padding: spacing.md, borderTopWidth: 1, borderTopColor: colors.border, backgroundColor: colors.backgroundElevated },
  input: { flex: 1, maxHeight: 120, minHeight: 44, paddingHorizontal: spacing.md, paddingVertical: spacing.sm, borderRadius: radii.lg, borderWidth: 1, borderColor: colors.border, backgroundColor: colors.background, color: colors.text, fontSize: 15 },
  send: { height: 44, paddingHorizontal: spacing.lg, borderRadius: radii.lg, backgroundColor: colors.primary, alignItems: "center", justifyContent: "center" },
  sendDisabled: { opacity: 0.45 }
});
