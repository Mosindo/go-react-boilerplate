import React, { useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { errorMessage } from "../api/errors";
import { chatApi, discoverApi } from "../api/services";
import type { Message, MessagesPage } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { ConfirmSheet, ReportSheet } from "../components/SafetySheets";
import { Sheet } from "../components/Sheet";
import { useToast } from "../components/Toast";
import { Button, EmptyState, ErrorState, Loader, Screen, Text } from "../components/ui";
import { fr } from "../i18n/fr";
import { dayKey, formatClock, formatDayLabel } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { useRealtime } from "../realtime/RealtimeProvider";
import { radius, spacing, useTheme } from "../theme";

type Item =
  | { kind: "message"; message: Message; showRead: boolean }
  | { kind: "day"; key: string; label: string };

export function ChatScreen({
  route,
  navigation
}: NativeStackScreenProps<RootStackParamList, "Chat">) {
  const { conversationId } = route.params;
  const { colors } = useTheme();
  const { user } = useAuth();
  const toast = useToast();
  const queryClient = useQueryClient();
  const realtime = useRealtime();
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const [menu, setMenu] = useState(false);
  const [confirm, setConfirm] = useState<"delete" | "unmatch" | null>(null);
  const [report, setReport] = useState(false);

  const key = useMemo(() => ["messages", conversationId] as const, [conversationId]);
  const query = useInfiniteQuery({
    queryKey: key,
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextBefore
  });

  const first = query.data?.pages[0];
  const peer = first?.peer;
  const messages = useMemo(() => query.data?.pages.flatMap((p) => p.messages) ?? [], [query.data]);

  useLayoutEffect(() => {
    navigation.setOptions({ title: peer?.firstName ?? route.params.name ?? "" });
  }, [navigation, peer?.firstName, route.params.name]);

  const markRead = useCallback(() => {
    chatApi
      .markRead(conversationId)
      .then(() => queryClient.invalidateQueries({ queryKey: ["conversations"] }))
      .catch(() => undefined);
  }, [conversationId, queryClient]);

  // Mark as read on open, and whenever a message arrives while this screen is on top.
  useEffect(() => {
    realtime.setActiveConversation(conversationId);
    const off = realtime.onMessageInActive(markRead);
    markRead();
    return () => {
      off();
      realtime.setActiveConversation(null);
    };
  }, [conversationId, markRead, realtime]);

  const items = useMemo<Item[]>(() => {
    // `messages` is newest-first, which is what an inverted list expects.
    const peerRead = first ? new Date(first.peerLastReadAt).getTime() : 0;
    const lastMine = messages.find((m) => m.senderId === user?.id);
    const out: Item[] = [];
    messages.forEach((message, i) => {
      out.push({
        kind: "message",
        message,
        showRead: message.id === lastMine?.id && new Date(message.createdAt).getTime() <= peerRead
      });
      const older = messages[i + 1];
      if (!older || dayKey(older.createdAt) !== dayKey(message.createdAt)) {
        out.push({
          kind: "day",
          key: dayKey(message.createdAt),
          label: formatDayLabel(message.createdAt)
        });
      }
    });
    return out;
  }, [messages, first, user?.id]);

  const send = async () => {
    const body = draft.trim();
    if (!body || sending) return;
    setSending(true);
    try {
      const message = await chatApi.send(conversationId, body);
      setDraft("");
      queryClient.setQueryData<InfiniteData<MessagesPage>>(key, (old) => {
        if (!old?.pages[0]) return old;
        const [head, ...rest] = old.pages;
        if (!head || head.messages.some((m) => m.id === message.id)) return old;
        return { ...old, pages: [{ ...head, messages: [message, ...head.messages] }, ...rest] };
      });
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
    } catch (error) {
      toast.show(errorMessage(error, fr.messages.sendFailed), "error");
    } finally {
      setSending(false);
    }
  };

  const leave = () => {
    void queryClient.invalidateQueries({ queryKey: ["conversations"] });
    queryClient.removeQueries({ queryKey: key });
    navigation.goBack();
  };

  const runConfirm = async () => {
    try {
      if (confirm === "delete") await chatApi.hide(conversationId);
      if (confirm === "unmatch") {
        const convs = queryClient.getQueryData<
          InfiniteData<{ conversations: { id: string; matchId: string }[] }>
        >(["conversations"]);
        const matchId = convs?.pages
          .flatMap((p) => p.conversations)
          .find((c) => c.id === conversationId)?.matchId;
        if (matchId) await discoverApi.unmatch(matchId);
      }
      setConfirm(null);
      leave();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  return (
    <Screen edges={["left", "right", "bottom"]}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        keyboardVerticalOffset={90}
      >
        <View
          style={{
            flexDirection: "row",
            justifyContent: "flex-end",
            paddingHorizontal: spacing.lg,
            paddingTop: spacing.xs
          }}
        >
          <Pressable
            testID="chat-menu"
            accessibilityRole="button"
            accessibilityLabel={fr.messages.options}
            onPress={() => setMenu(true)}
            hitSlop={10}
          >
            <Text variant="label" color={colors.primary}>
              ⋯ {fr.messages.options}
            </Text>
          </Pressable>
        </View>
        {query.isLoading ? <Loader /> : null}
        {query.isError ? <ErrorState onRetry={() => void query.refetch()} /> : null}
        {query.data && messages.length === 0 ? (
          <EmptyState
            glyph="👋"
            title={fr.messages.sayHi}
            body={fr.messages.chatEmpty(peer?.firstName ?? "")}
          />
        ) : null}
        {query.data && messages.length > 0 ? (
          <FlatList
            testID="message-list"
            inverted
            data={items}
            keyExtractor={(it) => (it.kind === "message" ? it.message.id : `day-${it.key}`)}
            contentContainerStyle={{ padding: spacing.lg, gap: spacing.sm }}
            onEndReached={() =>
              query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()
            }
            onEndReachedThreshold={0.3}
            ListFooterComponent={query.isFetchingNextPage ? <Loader /> : null}
            renderItem={({ item }) => {
              if (item.kind === "day") {
                return (
                  <Text
                    variant="caption"
                    muted
                    style={{ textAlign: "center", marginVertical: spacing.sm }}
                  >
                    {item.label}
                  </Text>
                );
              }
              const mine = item.message.senderId === user?.id;
              return (
                <View style={{ alignItems: mine ? "flex-end" : "flex-start" }}>
                  <View
                    testID={mine ? "bubble-mine" : "bubble-theirs"}
                    style={{
                      maxWidth: "80%",
                      paddingHorizontal: spacing.md,
                      paddingVertical: spacing.sm + 2,
                      borderRadius: radius.lg,
                      borderBottomRightRadius: mine ? 6 : radius.lg,
                      borderBottomLeftRadius: mine ? radius.lg : 6,
                      backgroundColor: mine ? colors.primary : colors.surface,
                      borderWidth: mine ? 0 : 1,
                      borderColor: colors.border
                    }}
                  >
                    <Text color={mine ? colors.primaryText : colors.text}>{item.message.body}</Text>
                  </View>
                  <Text variant="caption" muted style={{ marginTop: 2 }}>
                    {formatClock(item.message.createdAt)}
                    {item.showRead ? ` · ${fr.messages.read}` : ""}
                  </Text>
                </View>
              );
            }}
          />
        ) : (
          <View style={{ flex: 1 }} />
        )}
        <View
          style={{
            flexDirection: "row",
            alignItems: "flex-end",
            gap: spacing.sm,
            padding: spacing.md,
            borderTopWidth: 1,
            borderTopColor: colors.border
          }}
        >
          <TextInput
            testID="chat-input"
            accessibilityLabel={fr.messages.placeholder}
            value={draft}
            onChangeText={setDraft}
            placeholder={fr.messages.placeholder}
            placeholderTextColor={colors.textMuted}
            multiline
            maxLength={2000}
            style={{
              flex: 1,
              maxHeight: 120,
              minHeight: 44,
              color: colors.text,
              backgroundColor: colors.surface,
              borderColor: colors.border,
              borderWidth: 1,
              borderRadius: radius.lg,
              paddingHorizontal: spacing.lg,
              paddingVertical: spacing.sm + 2,
              fontSize: 16
            }}
          />
          <Button
            testID="chat-send"
            title={fr.messages.send}
            disabled={!draft.trim()}
            loading={sending}
            onPress={() => void send()}
          />
        </View>
      </KeyboardAvoidingView>

      <Sheet visible={menu} onClose={() => setMenu(false)} title={peer?.firstName}>
        <Button
          variant="secondary"
          title={fr.messages.deleteConversation}
          onPress={() => {
            setMenu(false);
            setConfirm("delete");
          }}
        />
        <Button
          variant="secondary"
          title={fr.messages.unmatch}
          onPress={() => {
            setMenu(false);
            setConfirm("unmatch");
          }}
        />
        <Button
          testID="chat-report"
          variant="secondary"
          title={fr.safety.report}
          onPress={() => {
            setMenu(false);
            setReport(true);
          }}
        />
        <Button variant="ghost" title={fr.common.close} onPress={() => setMenu(false)} />
      </Sheet>
      <ConfirmSheet
        visible={confirm !== null}
        title={confirm === "delete" ? fr.messages.deleteConversation : fr.messages.unmatch}
        message={confirm === "delete" ? fr.messages.deleteConfirm : fr.messages.unmatchConfirm}
        confirmLabel={fr.common.confirm}
        danger
        onConfirm={() => void runConfirm()}
        onClose={() => setConfirm(null)}
      />
      {peer ? (
        <ReportSheet
          visible={report}
          userId={peer.id}
          onClose={() => setReport(false)}
          onDone={(blocked) => {
            setReport(false);
            if (blocked) leave();
          }}
        />
      ) : null}
    </Screen>
  );
}
