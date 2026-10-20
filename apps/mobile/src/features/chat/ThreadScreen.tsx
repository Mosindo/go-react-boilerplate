import React, { useCallback, useEffect, useLayoutEffect, useState } from "react";
import { Alert, FlatList, StyleSheet, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { blockUser, fetchMessages, markConversationRead, sendMessage } from "../../api/social";
import { queryKeys } from "../../api/keys";
import type { ConversationsPage, Message, MessagesPage } from "../../api/types";
import { formatTime } from "../../lib/format";
import { MAX_MESSAGE_LENGTH, validateMessageBody } from "../../lib/validation";
import type { MainStackParamList } from "../../navigation/types";
import { clearConversationUnread, removeMessage, replaceMessage, upsertMessage } from "../../realtime/events";
import { setActiveConversation } from "../../realtime/activeConversation";
import { radii, spacing, useTheme } from "../../theme";
import { Button, ErrorState, LoadingState, Text, useToast } from "../../ui";
import { Screen } from "../../ui/Screen";
import { useAuth } from "../auth/AuthProvider";

type Props = NativeStackScreenProps<MainStackParamList, "Thread">;
type MessagesData = InfiniteData<MessagesPage, string | null>;

export function isReadByOther(message: Message, ownId: string, otherLastReadAt: string | null): boolean {
  return (
    message.senderId === ownId &&
    otherLastReadAt !== null &&
    new Date(message.createdAt) <= new Date(otherLastReadAt)
  );
}

export function ThreadScreen({ route, navigation }: Props) {
  const { conversationId, userId, name } = route.params;
  const { colors } = useTheme();
  const { user } = useAuth();
  const toast = useToast();
  const qc = useQueryClient();
  const [draft, setDraft] = useState("");
  const ownId = user?.id ?? "";

  const query = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => fetchMessages(conversationId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
  const messages = query.data?.pages.flatMap((p) => p.messages) ?? [];
  const otherLastReadAt = query.data?.pages[0]?.otherLastReadAt ?? null;
  const newestIncomingId = messages.find((m) => m.senderId !== ownId)?.id;
  const lastOwnId = messages.find((m) => m.senderId === ownId)?.id;

  useLayoutEffect(() => {
    navigation.setOptions({
      title: name,
      headerRight: () => (
        <Button
          label="More"
          variant="ghost"
          onPress={() =>
            Alert.alert(name, undefined, [
              { text: "View profile", onPress: () => navigation.navigate("ProfileDetail", { userId }) },
              { text: "Report", onPress: () => navigation.navigate("Report", { userId, name }) },
              {
                text: "Block",
                style: "destructive",
                onPress: () =>
                  void blockUser(userId)
                    .then(() =>
                      Promise.all([
                        qc.invalidateQueries({ queryKey: queryKeys.conversations }),
                        qc.invalidateQueries({ queryKey: queryKeys.matches })
                      ])
                    )
                    .then(() => navigation.navigate("Tabs"))
                    .catch((e: unknown) =>
                      toast.show(e instanceof Error ? e.message : "Could not block.", "error")
                    )
              },
              { text: "Cancel", style: "cancel" }
            ])
          }
        />
      )
    });
  }, [navigation, name, userId, qc, toast]);

  useEffect(() => {
    setActiveConversation(conversationId);
    return () => setActiveConversation(null);
  }, [conversationId]);

  const markRead = useCallback(() => {
    qc.setQueryData<InfiniteData<ConversationsPage, string | null>>(queryKeys.conversations, (d) =>
      clearConversationUnread(d, conversationId)
    );
    void markConversationRead(conversationId)
      .then(() => qc.invalidateQueries({ queryKey: queryKeys.notifications }))
      .catch(() => undefined);
  }, [conversationId, qc]);

  useEffect(() => {
    if (query.isSuccess) markRead();
  }, [query.isSuccess, newestIncomingId, markRead]);

  const send = useMutation({
    mutationFn: (body: string) => sendMessage(conversationId, body),
    onMutate: (body) => {
      const temp: Message = {
        id: `tmp-${Date.now()}`,
        conversationId,
        senderId: ownId,
        body,
        createdAt: new Date().toISOString()
      };
      qc.setQueryData<MessagesData>(queryKeys.messages(conversationId), (d) => upsertMessage(d, temp));
      return { tempId: temp.id };
    },
    onSuccess: (message, _body, context) => {
      qc.setQueryData<MessagesData>(queryKeys.messages(conversationId), (d) =>
        replaceMessage(d, context.tempId, message)
      );
      void qc.invalidateQueries({ queryKey: queryKeys.conversations });
    },
    onError: (error, body, context) => {
      if (context)
        qc.setQueryData<MessagesData>(queryKeys.messages(conversationId), (d) =>
          removeMessage(d, context.tempId)
        );
      setDraft(body);
      toast.show(error instanceof Error ? error.message : "Message not sent.", "error");
    }
  });

  const submit = () => {
    const problem = validateMessageBody(draft);
    if (problem) {
      if (draft.length > 0) toast.show(problem, "error");
      return;
    }
    const body = draft.trim();
    setDraft("");
    send.mutate(body);
  };

  if (query.isLoading) return <LoadingState />;
  if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;

  return (
    <Screen padded={false} edges={["bottom"]}>
      <FlatList
        inverted
        data={messages}
        keyExtractor={(m) => m.id}
        contentContainerStyle={{ padding: spacing.lg }}
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
        onEndReachedThreshold={0.4}
        ListEmptyComponent={
          <View style={{ transform: [{ scaleY: -1 }], padding: spacing.xl }}>
            <Text tone="muted" style={{ textAlign: "center" }}>
              You matched with {name}. Say hello!
            </Text>
          </View>
        }
        renderItem={({ item }) => {
          const own = item.senderId === ownId;
          return (
            <View style={[styles.bubbleRow, own ? styles.own : styles.other]}>
              <View
                style={[
                  styles.bubble,
                  { backgroundColor: own ? colors.primary : colors.surface, borderColor: colors.border }
                ]}
              >
                <Text style={{ color: own ? colors.onPrimary : colors.text }}>{item.body}</Text>
              </View>
              <Text variant="caption" tone="muted" style={styles.time}>
                {formatTime(item.createdAt)}
                {item.id === lastOwnId && isReadByOther(item, ownId, otherLastReadAt) ? " · Read" : ""}
                {item.id.startsWith("tmp-") ? " · Sending…" : ""}
              </Text>
            </View>
          );
        }}
      />
      <View style={[styles.composer, { borderTopColor: colors.border, backgroundColor: colors.background }]}>
        <TextInput
          testID="message-input"
          accessibilityLabel="Message"
          style={[
            styles.input,
            { color: colors.text, backgroundColor: colors.surface, borderColor: colors.border }
          ]}
          value={draft}
          onChangeText={setDraft}
          placeholder="Write a message"
          placeholderTextColor={colors.textMuted}
          multiline
          maxLength={MAX_MESSAGE_LENGTH}
        />
        <Button
          label="Send"
          onPress={submit}
          disabled={draft.trim().length === 0}
          testID="message-send"
          style={styles.send}
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  bubbleRow: { marginBottom: spacing.md, maxWidth: "82%" },
  own: { alignSelf: "flex-end", alignItems: "flex-end" },
  other: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: {
    borderRadius: radii.md,
    borderWidth: StyleSheet.hairlineWidth,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md
  },
  time: { marginTop: 2 },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    padding: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 120,
    borderWidth: 1,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    fontSize: 16
  },
  send: { marginLeft: spacing.sm, minHeight: 44, paddingHorizontal: spacing.lg }
});
