import React, { useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, Text, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { apiErrorMessage } from "../../api/client";
import { chatApi } from "../../api/endpoints";
import type { Message, Page } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";
import { SafetyMenu } from "../../components/SafetyMenu";
import { useToast } from "../../components/Toast";
import { EmptyState, ErrorState, Loading } from "../../components/ui";
import { formatClock } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { useRealtimeEvents } from "../../realtime/RealtimeContext";
import { fontSizes, radii, spacing, useTheme } from "../../theme/theme";

type Props = NativeStackScreenProps<MainStackParamList, "Chat">;
type Cache = InfiniteData<Page<Message>, number | undefined>;

export default function ChatScreen({ route, navigation }: Props) {
  const { matchId, partnerId, partnerName } = route.params;
  const { colors } = useTheme();
  const { user } = useAuth();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const [menu, setMenu] = useState(false);
  const key = useMemo(() => ["messages", matchId] as const, [matchId]);

  const query = useInfiniteQuery({
    queryKey: key,
    queryFn: ({ pageParam }) => chatApi.messages(matchId, pageParam),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (last) => (typeof last.nextCursor === "number" ? last.nextCursor : undefined)
  });
  const messages = useMemo(() => query.data?.pages.flatMap((p) => p.items) ?? [], [query.data]);

  /** Inserts a message at the top (newest) of the cache, ignoring duplicates. */
  const insert = useCallback(
    (message: Message) => {
      queryClient.setQueryData<Cache>(key, (old) => {
        if (!old) return old;
        if (old.pages.some((p) => p.items.some((m) => m.id === message.id))) return old;
        const [first, ...rest] = old.pages;
        if (!first) return old;
        return { ...old, pages: [{ ...first, items: [message, ...first.items] }, ...rest] };
      });
    },
    [key, queryClient]
  );

  const markRead = useCallback(() => {
    void chatApi
      .markRead(matchId)
      .then(() => {
        void queryClient.invalidateQueries({ queryKey: ["conversations"] });
        void queryClient.invalidateQueries({ queryKey: ["notifications"] });
      })
      .catch(() => undefined);
  }, [matchId, queryClient]);

  useEffect(() => {
    markRead();
  }, [markRead]);

  useRealtimeEvents((event) => {
    if (event.data.matchId !== matchId) return;
    if (event.type === "message.new") {
      const message = event.data as unknown as Message;
      insert(message);
      if (message.senderId !== user?.id) markRead();
    } else if (event.type === "message.read") {
      const readAt = String(event.data.readAt ?? new Date().toISOString());
      queryClient.setQueryData<Cache>(key, (old) =>
        old
          ? {
              ...old,
              pages: old.pages.map((p) => ({
                ...p,
                items: p.items.map((m) => (m.senderId === user?.id && !m.readAt ? { ...m, readAt } : m))
              }))
            }
          : old
      );
    } else if (event.type === "match.removed") {
      toast.show("Cette conversation n'existe plus.", "info");
      navigation.popToTop();
    }
  });

  useLayoutEffect(() => {
    navigation.setOptions({
      title: partnerName,
      headerRight: () => (
        <Pressable accessibilityRole="button" accessibilityLabel="Plus d'options" onPress={() => setMenu(true)} hitSlop={12}>
          <Text style={{ color: colors.text, fontSize: 24, fontWeight: "800" }}>⋯</Text>
        </Pressable>
      )
    });
  }, [navigation, partnerName, colors.text]);

  const send = async () => {
    const body = text.trim();
    if (!body || sending) return;
    setSending(true);
    try {
      const message = await chatApi.send(matchId, body);
      insert(message);
      setText("");
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
    } catch (e) {
      toast.show(apiErrorMessage(e), "error");
    } finally {
      setSending(false);
    }
  };

  const lastMineId = messages.find((m) => m.senderId === user?.id)?.id;

  return (
    <SafeAreaView edges={["bottom"]} style={{ flex: 1, backgroundColor: colors.background }}>
      <KeyboardAvoidingView
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        style={{ flex: 1 }}
        keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}
      >
        {query.isLoading ? <Loading /> : null}
        {query.isError ? <ErrorState message={apiErrorMessage(query.error)} onRetry={() => void query.refetch()} /> : null}
        {!query.isLoading && !query.isError ? (
          <FlatList
            inverted
            data={messages}
            keyExtractor={(m) => String(m.id)}
            contentContainerStyle={{ padding: spacing.lg, gap: 6 }}
            onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
            onEndReachedThreshold={0.3}
            ListEmptyComponent={
              <View style={{ transform: [{ scaleY: -1 }] }}>
                <EmptyState
                  emoji="👋"
                  title={`Dites bonjour à ${partnerName}`}
                  message="Un premier message simple et sincère fait toute la différence."
                />
              </View>
            }
            renderItem={({ item }) => {
              const mine = item.senderId === user?.id;
              return (
                <View style={{ alignItems: mine ? "flex-end" : "flex-start" }}>
                  <View
                    accessibilityLabel={`${mine ? "Vous" : partnerName} à ${formatClock(item.createdAt)} : ${item.body}`}
                    style={{
                      maxWidth: "80%",
                      backgroundColor: mine ? colors.bubbleMine : colors.bubbleTheirs,
                      borderRadius: radii.lg,
                      paddingHorizontal: spacing.md,
                      paddingVertical: spacing.sm
                    }}
                  >
                    <Text selectable style={{ color: mine ? colors.bubbleMineText : colors.text, fontSize: fontSizes.md }}>
                      {item.body}
                    </Text>
                  </View>
                  <Text style={{ color: colors.textMuted, fontSize: 11, marginTop: 2 }}>
                    {formatClock(item.createdAt)}
                    {mine && item.id === lastMineId && item.readAt ? " · Lu" : ""}
                  </Text>
                </View>
              );
            }}
          />
        ) : null}
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
            accessibilityLabel="Votre message"
            value={text}
            onChangeText={setText}
            placeholder="Votre message"
            placeholderTextColor={colors.textMuted}
            multiline
            maxLength={2000}
            testID="chat-input"
            style={{
              flex: 1,
              maxHeight: 120,
              minHeight: 44,
              borderRadius: radii.lg,
              borderWidth: 1.5,
              borderColor: colors.border,
              backgroundColor: colors.surface,
              color: colors.text,
              paddingHorizontal: spacing.md,
              paddingVertical: spacing.sm,
              fontSize: fontSizes.md
            }}
          />
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Envoyer"
            onPress={() => void send()}
            disabled={!text.trim() || sending}
            testID="chat-send"
            style={{
              width: 44,
              height: 44,
              borderRadius: 22,
              backgroundColor: colors.primary,
              alignItems: "center",
              justifyContent: "center",
              opacity: !text.trim() || sending ? 0.4 : 1
            }}
          >
            <Text style={{ color: colors.primaryText, fontSize: 20, fontWeight: "800" }}>➤</Text>
          </Pressable>
        </View>
      </KeyboardAvoidingView>
      <SafetyMenu
        visible={menu}
        onClose={() => setMenu(false)}
        userId={partnerId}
        name={partnerName}
        matchId={matchId}
        onGone={() => navigation.popToTop()}
        extraActions={[
          { label: "Voir le profil", onPress: () => navigation.navigate("UserProfile", { userId: partnerId, matchId }) },
          {
            label: "Masquer la conversation",
            onPress: () => {
              void chatApi.hide(matchId).then(() => {
                void queryClient.invalidateQueries({ queryKey: ["conversations"] });
                navigation.popToTop();
              });
            }
          }
        ]}
      />
    </SafeAreaView>
  );
}
