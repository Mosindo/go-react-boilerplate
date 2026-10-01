import React, { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, TextInput, View } from "react-native";
import { useFocusEffect } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { chatApi } from "../api/endpoints";
import { keys } from "../api/keys";
import type { Message } from "../api/types";
import { useAuth } from "../auth/AuthProvider";
import { useFeedback } from "../components/Feedback";
import { errorMessage } from "../components/forms";
import { EmptyState, ErrorState, IconButton, Loading, Screen, Text } from "../components/ui";
import { formatMessageTime } from "../lib/dates";
import type { RootStackParamList } from "../navigation/types";
import { radii, spacing, useTheme } from "../theme/theme";

type Props = NativeStackScreenProps<RootStackParamList, "Chat">;

const PAGE = 30;
const MAX_LEN = 2000;

type Pages = InfiniteData<Message[], string | undefined>;

export function ChatScreen({ navigation, route }: Props) {
  const { conversationId, userId, name } = route.params;
  const t = useTheme();
  const qc = useQueryClient();
  const { account } = useAuth();
  const { toast } = useFeedback();
  const [text, setText] = useState("");
  const inputRef = useRef<TextInput>(null);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: name,
      headerRight: () => (
        <IconButton icon="ellipsis-horizontal" label="Voir le profil et les options" onPress={() => navigation.navigate("ProfileDetail", { userId, conversationId })} testID="chat-menu" />
      )
    });
  }, [navigation, name, userId, conversationId]);

  // Newest-first pages from the API; the list is inverted so the newest message sits at the bottom.
  const query = useInfiniteQuery({
    queryKey: keys.messages(conversationId),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    getNextPageParam: (last) => (last.length === PAGE ? last[last.length - 1]?.createdAt : undefined)
  });
  const messages = query.data?.pages.flat() ?? [];

  const markRead = useMutation({
    mutationFn: () => chatApi.markRead(conversationId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.conversations });
      void qc.invalidateQueries({ queryKey: keys.unread });
    }
  });
  const markReadRef = useRef(markRead.mutate);
  markReadRef.current = markRead.mutate;

  const newestIncomingId = messages.find((m) => m.senderId !== account?.id && !m.readAt)?.id;
  useFocusEffect(
    useCallback(() => {
      if (newestIncomingId) markReadRef.current();
    }, [newestIncomingId])
  );
  useEffect(() => {
    if (newestIncomingId) markReadRef.current();
  }, [newestIncomingId]);

  const send = useMutation({
    mutationFn: (body: string) => chatApi.send(conversationId, body),
    onSuccess: (msg) => {
      qc.setQueryData<Pages>(keys.messages(conversationId), (old) => {
        if (!old) return old;
        const [first = [], ...rest] = old.pages;
        if (first.some((m) => m.id === msg.id)) return old;
        return { ...old, pages: [[msg, ...first], ...rest] };
      });
      void qc.invalidateQueries({ queryKey: keys.conversations });
    },
    onError: (e, body) => {
      setText(body);
      toast(errorMessage(e), "error");
    }
  });

  const submit = () => {
    const body = text.trim();
    if (!body || send.isPending) return;
    setText("");
    send.mutate(body);
    inputRef.current?.focus();
  };

  if (query.isLoading) return <Screen><Loading /></Screen>;
  if (query.error) return <Screen><ErrorState message={(query.error as Error).message} onRetry={() => void query.refetch()} /></Screen>;

  return (
    <Screen edges={["left", "right", "bottom"]} padded={false}>
      <KeyboardAvoidingView style={{ flex: 1 }} behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={90}>
        <FlatList
          inverted
          data={messages}
          keyExtractor={(m) => m.id}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          onEndReachedThreshold={0.3}
          contentContainerStyle={{ padding: spacing.lg, gap: spacing.sm, flexGrow: 1 }}
          ListFooterComponent={query.isFetchingNextPage ? <Loading /> : null}
          ListEmptyComponent={
            <View style={{ transform: [{ scaleY: -1 }], flex: 1 }}>
              <EmptyState icon="chatbubbles-outline" title={`Vous avez matché avec ${name}`} message="Envoyez le premier message pour briser la glace." />
            </View>
          }
          renderItem={({ item }) => {
            const mine = item.senderId === account?.id;
            return (
              <View style={{ alignItems: mine ? "flex-end" : "flex-start" }} testID={mine ? "message-mine" : "message-theirs"}>
                <View
                  style={{
                    maxWidth: "82%",
                    backgroundColor: mine ? t.primary : t.surface,
                    borderColor: mine ? t.primary : t.border,
                    borderWidth: 1,
                    borderRadius: radii.lg,
                    borderBottomRightRadius: mine ? 4 : radii.lg,
                    borderBottomLeftRadius: mine ? radii.lg : 4,
                    paddingVertical: spacing.sm,
                    paddingHorizontal: spacing.md
                  }}
                >
                  <Text color={mine ? t.onPrimary : t.text}>{item.body}</Text>
                </View>
                <Text variant="caption" muted style={{ marginTop: 2, marginHorizontal: 4 }}>
                  {formatMessageTime(item.createdAt)}
                  {mine && item.readAt ? " · Lu" : ""}
                </Text>
              </View>
            );
          }}
        />
        <View style={{ flexDirection: "row", alignItems: "flex-end", gap: spacing.sm, padding: spacing.md, borderTopWidth: 1, borderTopColor: t.border, backgroundColor: t.background }}>
          <TextInput
            ref={inputRef}
            value={text}
            onChangeText={setText}
            placeholder="Votre message…"
            placeholderTextColor={t.textMuted}
            accessibilityLabel="Votre message"
            multiline
            maxLength={MAX_LEN}
            onSubmitEditing={Platform.OS === "web" ? submit : undefined}
            testID="chat-input"
            style={{ flex: 1, minHeight: 44, maxHeight: 120, borderRadius: radii.lg, borderWidth: 1, borderColor: t.border, backgroundColor: t.surface, color: t.text, paddingHorizontal: spacing.lg, paddingTop: 12, paddingBottom: 12, fontSize: 16 }}
          />
          <IconButton
            icon="send"
            label="Envoyer"
            color={text.trim() ? t.primary : t.textMuted}
            onPress={submit}
            testID="chat-send"
          />
        </View>
      </KeyboardAvoidingView>
    </Screen>
  );
}
