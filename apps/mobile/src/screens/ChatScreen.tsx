import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import {
  Alert,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  TextInput,
  View
} from "react-native";
import { useFocusEffect } from "@react-navigation/native";
import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
  type InfiniteData
} from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { chatApi } from "../api/endpoints";
import type { Message, MessagePage } from "../api/types";
import { formatTime } from "../lib/format";
import type { MainScreenProps } from "../navigation/types";
import { keys } from "../state/keys";
import { mergeMessage, useRealtime } from "../state/realtime";
import { useSession } from "../state/session";
import { radii, usePalette } from "../theme";
import { Screen } from "../ui/Screen";
import { EmptyState, ErrorState, Loading } from "../ui/states";
import { Text } from "../ui/Text";
import { useToast } from "../ui/Toast";

const MAX_LENGTH = 2000;

export function ChatScreen({ navigation, route }: MainScreenProps<"Chat">) {
  const { conversationId, peer, matchId } = route.params;
  const { user } = useSession();
  const p = usePalette();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { setActiveConversation } = useRealtime();
  const [draft, setDraft] = useState("");
  const listRef = useRef<FlatList<Message>>(null);

  const query = useInfiniteQuery({
    queryKey: keys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
    retry: (count, error) => !(error instanceof ApiError && error.status === 404) && count < 2
  });

  const messages = query.data?.pages.flatMap((page) => page.messages) ?? [];
  const otherLastReadId = query.data?.pages[0]?.otherLastReadId ?? 0;
  const newestPeerMessageId = messages.find((m) => m.senderId !== user?.id)?.id ?? 0;

  useFocusEffect(
    useCallback(() => {
      setActiveConversation(conversationId);
      return () => setActiveConversation(null);
    }, [conversationId, setActiveConversation])
  );

  // Mark the conversation as read whenever a new peer message is on screen.
  const markRead = useMutation({
    mutationFn: () => chatApi.markRead(conversationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      void queryClient.invalidateQueries({ queryKey: keys.notifications });
    }
  });
  const markReadMutate = markRead.mutate;
  useEffect(() => {
    if (newestPeerMessageId > 0) markReadMutate();
  }, [newestPeerMessageId, markReadMutate]);

  const send = useMutation({
    mutationFn: (body: string) => chatApi.send(conversationId, body),
    onSuccess: (message) => {
      queryClient.setQueryData<InfiniteData<MessagePage>>(keys.messages(conversationId), (d) =>
        mergeMessage(d, message)
      );
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
    },
    onError: (e: Error, body) => {
      setDraft((current) => current || body);
      toast.show(e.message, "error");
    }
  });

  const submit = () => {
    const body = draft.trim();
    if (!body || send.isPending) return;
    setDraft("");
    send.mutate(body);
  };

  useLayoutEffect(() => {
    navigation.setOptions({
      title: peer.firstName,
      headerRight: () => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Conversation options"
          hitSlop={12}
          onPress={() =>
            Alert.alert(peer.firstName, undefined, [
              {
                text: "View profile",
                onPress: () =>
                  navigation.navigate("UserProfile", {
                    userId: peer.userId,
                    conversationId,
                    matchId
                  })
              },
              {
                text: "Delete chat for me",
                style: "destructive",
                onPress: () =>
                  chatApi
                    .clear(conversationId)
                    .then(() => {
                      void queryClient.invalidateQueries({ queryKey: keys.conversations });
                      navigation.goBack();
                    })
                    .catch((e: Error) => toast.show(e.message, "error"))
              },
              { text: "Cancel", style: "cancel" }
            ])
          }
        >
          <Text variant="heading">⋯</Text>
        </Pressable>
      )
    });
  }, [navigation, peer, conversationId, matchId, queryClient, toast]);

  if (query.isLoading)
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  if (query.isError) {
    const gone = query.error instanceof ApiError && query.error.status === 404;
    return (
      <Screen>
        {gone ? (
          <EmptyState
            title="Conversation unavailable"
            message="This match is no longer active."
            actionLabel="Back"
            onAction={() => navigation.goBack()}
          />
        ) : (
          <ErrorState message={query.error.message} onRetry={() => void query.refetch()} />
        )}
      </Screen>
    );
  }

  return (
    <Screen edges={["left", "right", "bottom"]}>
      <KeyboardAvoidingView
        style={styles.fill}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}
      >
        <FlatList
          ref={listRef}
          inverted
          data={messages}
          keyExtractor={(m) => String(m.id)}
          contentContainerStyle={styles.list}
          onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
          ListEmptyComponent={
            <View style={styles.invertedEmpty}>
              <Text tone="muted" style={styles.center}>
                You matched with {peer.firstName}. Say hello!
              </Text>
            </View>
          }
          renderItem={({ item }) => {
            const mine = item.senderId === user?.id;
            return (
              <View style={[styles.bubbleRow, mine ? styles.mine : styles.theirs]}>
                <View
                  style={[
                    styles.bubble,
                    {
                      backgroundColor: mine ? p.primary : p.surface,
                      borderColor: p.border,
                      borderWidth: mine ? 0 : 1
                    }
                  ]}
                >
                  <Text style={{ color: mine ? p.primaryText : p.text }}>{item.body}</Text>
                </View>
                <Text variant="caption" tone="muted" style={styles.meta}>
                  {formatTime(item.createdAt)}
                  {mine ? (item.id <= otherLastReadId ? " · Read" : " · Sent") : ""}
                </Text>
              </View>
            );
          }}
        />
        <View
          style={[styles.composer, { borderTopColor: p.border, backgroundColor: p.background }]}
        >
          <TextInput
            testID="chat-input"
            accessibilityLabel="Message"
            value={draft}
            onChangeText={setDraft}
            placeholder="Write a message"
            placeholderTextColor={p.textMuted}
            multiline
            maxLength={MAX_LENGTH}
            style={[
              styles.input,
              { color: p.text, backgroundColor: p.surface, borderColor: p.border }
            ]}
          />
          <Pressable
            testID="chat-send"
            accessibilityRole="button"
            accessibilityLabel="Send message"
            onPress={submit}
            disabled={!draft.trim() || send.isPending}
            style={[
              styles.send,
              { backgroundColor: p.primary, opacity: !draft.trim() || send.isPending ? 0.5 : 1 }
            ]}
          >
            <Text style={{ color: p.primaryText, fontWeight: "800" }}>↑</Text>
          </Pressable>
        </View>
      </KeyboardAvoidingView>
    </Screen>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  list: { paddingHorizontal: 16, paddingVertical: 12, gap: 8 },
  invertedEmpty: { transform: [{ scaleY: -1 }], padding: 24 },
  center: { textAlign: "center" },
  bubbleRow: { maxWidth: "82%", gap: 2 },
  mine: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { paddingHorizontal: 14, paddingVertical: 10, borderRadius: radii.lg },
  meta: { paddingHorizontal: 4 },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: 8,
    padding: 10,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    borderWidth: 1,
    borderRadius: radii.lg,
    paddingHorizontal: 14,
    paddingTop: 10,
    paddingBottom: 10,
    fontSize: 16
  },
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" }
});
