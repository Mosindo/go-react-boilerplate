import React, { useEffect, useMemo, useRef, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { ApiError } from "../../api/client";
import { chatApi, matchingApi } from "../../api/endpoints";
import type { Message, MessagesPage } from "../../api/types";
import { errorMessage, formatClock, formatDayLabel } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { useAuth } from "../../providers/AuthProvider";
import { keys } from "../../providers/queryClient";
import { useRealtimeConnected } from "../../providers/RealtimeProvider";
import { useTheme } from "../../theme";
import { ActionSheet, Avatar, EmptyState, ErrorState, Header, IconButton, LoadingState, Screen, showToast, Text } from "../../ui";
import { useBlock } from "../safety/useBlock";

type Props = NativeStackScreenProps<AppStackParamList, "Chat">;

type Row = { kind: "message"; message: Message; pending?: boolean } | { kind: "day"; id: string; label: string };

export function ChatScreen({ navigation, route }: Props) {
  const { conversationId } = route.params;
  const { user } = useAuth();
  const { colors, radii, spacing, typography } = useTheme();
  const queryClient = useQueryClient();
  const live = useRealtimeConnected();
  const [draft, setDraft] = useState("");
  const [pending, setPending] = useState<Message[]>([]);
  const [menu, setMenu] = useState(false);
  const block = useBlock(() => navigation.goBack());
  const inputRef = useRef<TextInput>(null);

  const conversation = useQuery({ queryKey: keys.conversation(conversationId), queryFn: () => chatApi.get(conversationId) });
  const messages = useInfiniteQuery({
    queryKey: keys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam, 30),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => (last.hasMore ? last.messages[last.messages.length - 1]?.id : undefined),
    refetchInterval: live ? false : 8_000
  });

  const all = useMemo(() => messages.data?.pages.flatMap((p) => p.messages) ?? [], [messages.data]);
  const otherLastReadAt = messages.data?.pages[0]?.otherLastReadAt ?? null;
  const newestIncoming = all.find((m) => m.senderId !== user?.id)?.id;

  // Mark as read when opening the thread and whenever a new message arrives.
  useEffect(() => {
    if (!newestIncoming && !messages.isSuccess) {
      return;
    }
    chatApi
      .markRead(conversationId)
      .then(() => {
        void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
        void queryClient.invalidateQueries({ queryKey: keys.notifications });
      })
      .catch(() => undefined);
  }, [conversationId, newestIncoming, messages.isSuccess, queryClient]);

  const send = useMutation({
    mutationFn: (body: string) => chatApi.send(conversationId, body),
    onMutate: (body) => {
      const temp: Message = { id: `pending-${Date.now()}`, conversationId, senderId: user?.id ?? "", body, createdAt: new Date().toISOString() };
      setPending((p) => [temp, ...p]);
      setDraft("");
      return { temp };
    },
    onSuccess: (message, _, context) => {
      setPending((p) => p.filter((m) => m.id !== context?.temp.id));
      queryClient.setQueryData<InfiniteData<MessagesPage>>(keys.messages(conversationId), (data) => {
        if (!data || data.pages.some((page) => page.messages.some((m) => m.id === message.id))) {
          return data;
        }
        const [first, ...rest] = data.pages;
        return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
      });
      void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
    },
    onError: (error, body, context) => {
      setPending((p) => p.filter((m) => m.id !== context?.temp.id));
      setDraft(body);
      showToast(errorMessage(error), "error");
    }
  });

  const clear = useMutation({
    mutationFn: () => chatApi.clear(conversationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.messages(conversationId) });
      void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
      showToast("Conversation effacée de ton côté.", "success");
      navigation.goBack();
    },
    onError: (e) => showToast(errorMessage(e), "error")
  });
  const unmatch = useMutation({
    mutationFn: (matchId: string) => matchingApi.unmatch(matchId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
      showToast("Match annulé.", "success");
      navigation.goBack();
    },
    onError: (e) => showToast(errorMessage(e), "error")
  });

  const rows = useMemo<Row[]>(() => {
    // The list is inverted: index 0 is the newest message.
    const merged = [...pending.map((m) => ({ message: m, pending: true })), ...all.map((m) => ({ message: m, pending: false }))];
    const out: Row[] = [];
    merged.forEach((item, i) => {
      out.push({ kind: "message", ...item });
      const next = merged[i + 1];
      const day = formatDayLabel(item.message.createdAt);
      if (!next || formatDayLabel(next.message.createdAt) !== day) {
        out.push({ kind: "day", id: `day-${item.message.id}`, label: day });
      }
    });
    return out;
  }, [all, pending]);

  const gone =
    (conversation.error instanceof ApiError && conversation.error.status === 404) || (messages.error instanceof ApiError && messages.error.status === 404);
  const other = conversation.data?.user;
  const lastMineId = all.find((m) => m.senderId === user?.id)?.id;

  const submit = () => {
    const body = draft.trim();
    if (body && !send.isPending) {
      send.mutate(body);
    }
  };

  return (
    <Screen edges={["top", "left", "right", "bottom"]}>
      <Header
        back
        right={
          other ? <IconButton icon="ellipsis-horizontal" label="Options de la conversation" onPress={() => setMenu(true)} testID="chat-menu" /> : undefined
        }
        subtitle={other ? `${other.age} ans` : undefined}
        title={other?.firstName ?? "Conversation"}
      />
      {gone ? (
        <EmptyState icon="heart-dislike-outline" message="Ce match a pris fin. La conversation n’est plus accessible." title="Conversation indisponible" />
      ) : messages.isLoading ? (
        <LoadingState />
      ) : messages.isError ? (
        <ErrorState message={errorMessage(messages.error)} onRetry={() => void messages.refetch()} />
      ) : (
        <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={60} style={styles.flex}>
          <FlatList
            contentContainerStyle={{ padding: spacing.md, gap: 6, flexGrow: 1 }}
            data={rows}
            inverted={rows.length > 0}
            keyExtractor={(row) => (row.kind === "day" ? row.id : row.message.id)}
            ListEmptyComponent={
              other ? (
                <View style={[styles.flex, { alignItems: "center", justifyContent: "center", gap: spacing.sm, padding: spacing.xl }]}>
                  <Avatar name={other.firstName} ring size={88} uri={other.photoUrl} />
                  <Text center variant="heading">
                    Vous avez matché !
                  </Text>
                  <Text center tone="muted">
                    Brise la glace avec {other.firstName} : une question sur ses centres d’intérêt marche souvent très bien.
                  </Text>
                </View>
              ) : null
            }
            onEndReached={() => {
              if (messages.hasNextPage && !messages.isFetchingNextPage) {
                void messages.fetchNextPage();
              }
            }}
            onEndReachedThreshold={0.3}
            renderItem={({ item }) => {
              if (item.kind === "day") {
                return (
                  <Text center style={{ marginVertical: spacing.xs }} tone="subtle" variant="caption">
                    {item.label}
                  </Text>
                );
              }
              const mine = item.message.senderId === user?.id;
              const seen =
                mine &&
                item.message.id === lastMineId &&
                !!otherLastReadAt &&
                new Date(otherLastReadAt).getTime() >= new Date(item.message.createdAt).getTime();
              return (
                <View style={{ alignItems: mine ? "flex-end" : "flex-start" }}>
                  <View
                    style={{
                      maxWidth: "80%",
                      paddingHorizontal: 14,
                      paddingVertical: 9,
                      borderRadius: radii.lg,
                      borderBottomRightRadius: mine ? 6 : radii.lg,
                      borderBottomLeftRadius: mine ? radii.lg : 6,
                      backgroundColor: mine ? colors.primary : colors.surfaceAlt,
                      opacity: item.pending ? 0.6 : 1
                    }}
                    testID={mine ? "message-mine" : "message-theirs"}
                  >
                    <Text style={{ color: mine ? colors.onPrimary : colors.text }}>{item.message.body}</Text>
                  </View>
                  <Text style={{ marginTop: 2, marginHorizontal: 6 }} tone="subtle" variant="caption">
                    {item.pending ? "Envoi…" : formatClock(new Date(item.message.createdAt))}
                    {mine && item.message.id === lastMineId && !item.pending ? (seen ? " · Vu" : " · Envoyé") : ""}
                  </Text>
                </View>
              );
            }}
          />
          <View style={[styles.composer, { borderTopColor: colors.border, padding: spacing.sm, gap: spacing.xs, backgroundColor: colors.background }]}>
            <TextInput
              accessibilityLabel="Écrire un message"
              maxLength={2000}
              multiline
              onChangeText={setDraft}
              onKeyPress={(e) => {
                const native = e.nativeEvent as { key: string; shiftKey?: boolean };
                if (Platform.OS === "web" && native.key === "Enter" && !native.shiftKey) {
                  e.preventDefault();
                  submit();
                }
              }}
              placeholder="Ton message…"
              placeholderTextColor={colors.textSubtle}
              ref={inputRef}
              style={[
                typography.body,
                styles.input,
                { color: colors.text, backgroundColor: colors.surface, borderColor: colors.border, borderRadius: radii.lg }
              ]}
              testID="chat-input"
              value={draft}
            />
            <Pressable
              accessibilityLabel="Envoyer"
              accessibilityRole="button"
              disabled={!draft.trim()}
              onPress={submit}
              style={[styles.send, { backgroundColor: draft.trim() ? colors.primary : colors.surfaceAlt }]}
              testID="chat-send"
            >
              <Text style={{ color: draft.trim() ? colors.onPrimary : colors.textSubtle }} variant="label">
                Envoyer
              </Text>
            </Pressable>
          </View>
        </KeyboardAvoidingView>
      )}
      <ActionSheet
        actions={
          other && conversation.data
            ? [
                { label: `Voir le profil de ${other.firstName}`, onPress: () => navigation.navigate("ProfileDetail", { userId: other.userId }) },
                { label: "Effacer la conversation (pour moi)", onPress: () => clear.mutate() },
                { label: "Annuler le match", destructive: true, onPress: () => unmatch.mutate(conversation.data.matchId) },
                { label: `Bloquer ${other.firstName}`, destructive: true, onPress: () => block.mutate(other.userId) },
                {
                  label: `Signaler ${other.firstName}`,
                  destructive: true,
                  onPress: () => navigation.navigate("Report", { userId: other.userId, firstName: other.firstName })
                }
              ]
            : []
        }
        onClose={() => setMenu(false)}
        title="Conversation"
        visible={menu}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  composer: { flexDirection: "row", alignItems: "flex-end", borderTopWidth: StyleSheet.hairlineWidth },
  input: { flex: 1, minHeight: 44, maxHeight: 120, paddingHorizontal: 14, paddingTop: 10, paddingBottom: 10, borderWidth: 1 },
  send: { height: 44, paddingHorizontal: 16, borderRadius: 22, alignItems: "center", justifyContent: "center" }
});
