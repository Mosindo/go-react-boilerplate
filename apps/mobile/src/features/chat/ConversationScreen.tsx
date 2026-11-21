import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ActivityIndicator, FlatList, KeyboardAvoidingView, Platform, Pressable, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useFocusEffect } from "@react-navigation/native";
import { useInfiniteQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { ApiError, errorMessage } from "../../api/client";
import { chatApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { ChatMessage, ChatSummary, MessagesPage } from "../../api/types";
import type { RootStackParamList } from "../../navigation/types";
import { setActiveConversation } from "../../realtime/activeConversation";
import { makeStyles, useTheme } from "../../theme";
import { Avatar, EmptyState, ErrorState, Header, IconButton, LoadingState, showToast, Text } from "../../ui";
import { formatDayLabel, formatTime } from "../../utils/format";
import { useAuth } from "../auth/AuthProvider";
import { SafetyMenu, type SafetyMenuHandle } from "../safety/SafetyMenu";

type Props = NativeStackScreenProps<RootStackParamList, "Conversation">;

type Row = { kind: "message"; message: ChatMessage; showRead: boolean } | { kind: "day"; id: string; label: string };

export function ConversationScreen({ navigation, route }: Props) {
  const styles = useStyles();
  const theme = useTheme();
  const { userId } = route.params;
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const safetyRef = useRef<SafetyMenuHandle>(null);
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);

  const summary = useMemo(() => {
    const data = queryClient.getQueryData<InfiniteData<ChatSummary[]>>(queryKeys.chats);
    return data?.pages.flat().find((c) => c.user.id === userId);
  }, [queryClient, userId]);
  const firstName = route.params.firstName ?? summary?.user.firstName ?? "Conversation";
  const photoUrl = route.params.photoUrl ?? summary?.user.photoUrl ?? null;

  const messages = useInfiniteQuery({
    queryKey: queryKeys.messages(userId),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => chatApi.messages(userId, pageParam),
    getNextPageParam: (last: MessagesPage) => (last.hasMore ? last.messages[0]?.id : undefined),
    retry: (count, err) => !(err instanceof ApiError && err.status === 404) && count < 2
  });

  useFocusEffect(
    useCallback(() => {
      setActiveConversation(userId);
      return () => setActiveConversation(null);
    }, [userId])
  );

  const chronological = useMemo(() => {
    const pages = messages.data?.pages ?? [];
    return [...pages].reverse().flatMap((p) => p.messages);
  }, [messages.data]);

  // Mark as read whenever a message from the other member is the latest one.
  const latestIncoming = [...chronological].reverse().find((m) => m.senderUserId !== user?.id);
  useEffect(() => {
    if (!latestIncoming) return;
    void chatApi
      .markRead(userId)
      .then(() => queryClient.invalidateQueries({ queryKey: queryKeys.chats }))
      .catch(() => undefined);
  }, [latestIncoming?.id, userId, queryClient]); // eslint-disable-line react-hooks/exhaustive-deps

  const rows = useMemo<Row[]>(() => {
    const out: Row[] = [];
    let lastDay = "";
    const lastMineId = [...chronological].reverse().find((m) => m.senderUserId === user?.id)?.id;
    chronological.forEach((m) => {
      const day = new Date(m.createdAt).toDateString();
      if (day !== lastDay) {
        out.push({ kind: "day", id: `day-${day}`, label: formatDayLabel(m.createdAt) });
        lastDay = day;
      }
      out.push({ kind: "message", message: m, showRead: m.id === lastMineId });
    });
    return out.reverse(); // inverted list: newest first
  }, [chronological, user?.id]);

  const send = async () => {
    const content = draft.trim();
    if (!content || sending) return;
    setSending(true);
    setDraft("");
    try {
      const message = await chatApi.send(userId, content);
      queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(userId), (data) => {
        if (!data || data.pages.length === 0) {
          return { pages: [{ messages: [message], hasMore: false, otherLastReadAt: null }], pageParams: [undefined] };
        }
        const [first, ...rest] = data.pages;
        if (first.messages.some((m) => m.id === message.id)) return data;
        return { ...data, pages: [{ ...first, messages: [...first.messages, message] }, ...rest] };
      });
      void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
    } catch (err) {
      setDraft(content);
      showToast(errorMessage(err), "error");
    } finally {
      setSending(false);
    }
  };

  const unavailable = messages.isError && messages.error instanceof ApiError && messages.error.status === 404;

  return (
    <SafeAreaView style={styles.root} edges={["top", "bottom"]}>
      <Header
        center={
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`Voir le profil de ${firstName}`}
            onPress={() => navigation.navigate("ProfileDetail", { userId, mode: "match" })}
            style={styles.titleRow}
          >
            <Avatar uri={photoUrl} name={firstName} size={34} />
            <Text variant="heading" numberOfLines={1}>
              {firstName}
            </Text>
          </Pressable>
        }
        right={
          <IconButton
            testID="conversation-menu"
            icon="ellipsis-horizontal"
            accessibilityLabel="Options"
            onPress={() => safetyRef.current?.open()}
          />
        }
      />
      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={8}>
        {messages.isLoading ? (
          <LoadingState />
        ) : unavailable ? (
          <EmptyState
            icon="chatbubbles-outline"
            title="Conversation indisponible"
            message="Ce match n'existe plus."
            actionLabel="Retour"
            onAction={() => navigation.goBack()}
          />
        ) : messages.isError ? (
          <ErrorState message={errorMessage(messages.error)} onRetry={() => void messages.refetch()} />
        ) : (
          <FlatList
            testID="conversation-messages"
            inverted={rows.length > 0}
            data={rows}
            keyExtractor={(r) => (r.kind === "day" ? r.id : r.message.id)}
            contentContainerStyle={styles.messages}
            onEndReached={() => {
              if (messages.hasNextPage && !messages.isFetchingNextPage) void messages.fetchNextPage();
            }}
            onEndReachedThreshold={0.3}
            ListFooterComponent={messages.isFetchingNextPage ? <ActivityIndicator color={theme.colors.primary} /> : null}
            ListEmptyComponent={
              <View style={styles.empty}>
                <Avatar uri={photoUrl} name={firstName} size={96} ring />
                <Text variant="title" align="center">
                  Vous avez matché avec {firstName}
                </Text>
                <Text tone="muted" align="center">
                  Un simple bonjour suffit souvent. Parlez de ce qui vous a plu dans son profil.
                </Text>
              </View>
            }
            renderItem={({ item }) => {
              if (item.kind === "day") {
                return (
                  <Text variant="caption" tone="subtle" align="center" style={styles.day}>
                    {item.label}
                  </Text>
                );
              }
              const mine = item.message.senderUserId === user?.id;
              return (
                <View style={[styles.bubbleRow, mine ? styles.mineRow : null]}>
                  <View
                    style={[styles.bubble, mine ? styles.mine : styles.theirs]}
                    accessible
                    accessibilityLabel={`${mine ? "Vous" : firstName}, ${formatTime(item.message.createdAt)} : ${item.message.content}`}
                  >
                    <Text tone={mine ? "onPrimary" : "default"}>{item.message.content}</Text>
                    <Text variant="caption" tone={mine ? "onPrimary" : "subtle"} style={styles.time}>
                      {formatTime(item.message.createdAt)}
                    </Text>
                  </View>
                  {mine && item.showRead ? (
                    <Text testID="message-read-state" variant="caption" tone="subtle" style={styles.readState}>
                      {item.message.isRead ? "Lu" : "Envoyé"}
                    </Text>
                  ) : null}
                </View>
              );
            }}
          />
        )}
        {!unavailable ? (
          <View style={styles.composer}>
            <TextInput
              testID="conversation-input"
              accessibilityLabel="Votre message"
              value={draft}
              onChangeText={setDraft}
              placeholder="Écrire un message…"
              placeholderTextColor={theme.colors.textSubtle}
              multiline
              maxLength={2000}
              style={styles.input}
              onKeyPress={(e) => {
                const native = e.nativeEvent as { key: string; shiftKey?: boolean };
                if (Platform.OS === "web" && native.key === "Enter" && !native.shiftKey) {
                  e.preventDefault();
                  void send();
                }
              }}
            />
            <IconButton
              testID="conversation-send"
              icon="arrow-up"
              accessibilityLabel="Envoyer"
              filled
              tone="primary"
              size={44}
              disabled={!draft.trim() || sending}
              onPress={() => void send()}
            />
          </View>
        ) : null}
      </KeyboardAvoidingView>
      <SafetyMenu ref={safetyRef} userId={userId} firstName={firstName} matched allowClear onRemoved={() => navigation.goBack()} />
    </SafeAreaView>
  );
}

const useStyles = makeStyles((t) => ({
  root: { flex: 1, backgroundColor: t.colors.background },
  flex: { flex: 1, width: "100%", maxWidth: 720, alignSelf: "center" },
  titleRow: { flexDirection: "row", alignItems: "center", gap: t.spacing.sm, maxWidth: "100%" },
  messages: { paddingHorizontal: t.spacing.lg, paddingVertical: t.spacing.md, gap: t.spacing.xs, flexGrow: 1 },
  empty: { flex: 1, alignItems: "center", justifyContent: "center", gap: t.spacing.md, padding: t.spacing.xl },
  day: { marginVertical: t.spacing.md },
  bubbleRow: { alignItems: "flex-start" },
  mineRow: { alignItems: "flex-end" },
  bubble: { maxWidth: "80%", paddingHorizontal: t.spacing.md, paddingVertical: t.spacing.sm, borderRadius: t.radii.lg },
  mine: { backgroundColor: t.colors.primary, borderBottomRightRadius: t.radii.sm },
  theirs: { backgroundColor: t.colors.surface, borderWidth: 1, borderColor: t.colors.border, borderBottomLeftRadius: t.radii.sm },
  time: { opacity: 0.75, alignSelf: "flex-end", fontSize: 11 },
  readState: { marginTop: 2, marginRight: t.spacing.xs },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: t.spacing.sm,
    paddingHorizontal: t.spacing.lg,
    paddingVertical: t.spacing.sm,
    borderTopWidth: 1,
    borderTopColor: t.colors.border,
    backgroundColor: t.colors.background
  },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 140,
    borderRadius: t.radii.lg,
    borderWidth: 1,
    borderColor: t.colors.border,
    backgroundColor: t.colors.surface,
    paddingHorizontal: t.spacing.lg,
    paddingTop: 11,
    paddingBottom: 11,
    color: t.colors.text,
    fontSize: 16
  }
}));
