import React, { useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { FlatList, StyleSheet, TextInput, View } from "react-native";
import { useInfiniteQuery, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { useIsFocused } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { ActionSheet, Avatar, ErrorState, IconButton, LoadingState, Screen, Text } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { errorMessage } from "../../lib/api/client";
import { chatApi, discoveryApi } from "../../lib/api/endpoints";
import type { Message, MessagesPage } from "../../lib/api/types";
import { formatMessageTime } from "../../lib/format";
import { queryKeys } from "../../lib/queryClient";
import { useRealtime } from "../../lib/realtime/RealtimeProvider";
import { useSession } from "../../lib/session/SessionProvider";
import { showToast } from "../../lib/toast";
import type { AppStackParamList } from "../../navigation/types";
import { SafetySheet } from "../safety/SafetySheet";

type Props = NativeStackScreenProps<AppStackParamList, "Chat">;
type PendingMessage = Message & { pending?: boolean; failed?: boolean };

const MAX_LENGTH = 2000;

export default function ChatScreen({ route, navigation }: Props) {
  const { conversationId } = route.params;
  const { colors, radii } = useTheme();
  const { userId } = useSession();
  const { connected } = useRealtime();
  const client = useQueryClient();
  const focused = useIsFocused();
  const [draft, setDraft] = useState("");
  const [pending, setPending] = useState<PendingMessage[]>([]);
  const [menuOpen, setMenuOpen] = useState(false);
  const [confirm, setConfirm] = useState<"unmatch" | "hide" | null>(null);
  const [safetyOpen, setSafetyOpen] = useState(false);

  const conversation = useQuery({ queryKey: queryKeys.conversation(conversationId), queryFn: () => chatApi.conversation(conversationId) });
  const messages = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    refetchInterval: connected ? false : 5_000
  });

  const other = conversation.data?.user;
  const serverMessages = useMemo(() => messages.data?.pages.flatMap((p) => p.messages) ?? [], [messages.data]);
  const otherLastReadAt = messages.data?.pages[0]?.otherLastReadAt ?? null;
  const items: PendingMessage[] = useMemo(() => {
    const ids = new Set(serverMessages.map((m) => m.id));
    return [...pending.filter((p) => !ids.has(p.id)).reverse(), ...serverMessages];
  }, [pending, serverMessages]);

  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <View style={styles.headerTitle}>
          <Avatar name={other?.firstName ?? route.params.name ?? ""} size={32} uri={other?.photo?.url} />
          <Text variant="label">{other?.firstName ?? route.params.name ?? ""}</Text>
        </View>
      ),
      headerRight: () => <IconButton icon="ellipsis-horizontal" label="Options de la conversation" onPress={() => setMenuOpen(true)} testID="chat-menu" />
    });
  }, [navigation, other, route.params.name]);

  // Mark as read whenever the thread is visible and a new message from the other person arrives.
  const newestFromOther = serverMessages.find((m) => m.senderId !== userId)?.id;
  useEffect(() => {
    if (!focused || !newestFromOther) return;
    chatApi
      .markRead(conversationId)
      .then(() => client.invalidateQueries({ queryKey: queryKeys.conversations }))
      .catch(() => undefined);
  }, [client, conversationId, focused, newestFromOther]);

  const send = useCallback(async () => {
    const body = draft.trim();
    if (!body) return;
    const temp: PendingMessage = {
      id: `pending-${Date.now()}`,
      conversationId,
      senderId: userId ?? "",
      body,
      createdAt: new Date().toISOString(),
      pending: true
    };
    setDraft("");
    setPending((p) => [...p, temp]);
    try {
      const saved = await chatApi.send(conversationId, body);
      client.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conversationId), (data) => {
        if (!data?.pages.length) return data;
        const [first, ...rest] = data.pages;
        if (first.messages.some((m) => m.id === saved.id)) return data;
        return { ...data, pages: [{ ...first, messages: [saved, ...first.messages] }, ...rest] };
      });
      setPending((p) => p.filter((m) => m.id !== temp.id));
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
    } catch (e) {
      setPending((p) => p.map((m) => (m.id === temp.id ? { ...m, pending: false, failed: true } : m)));
      showToast(errorMessage(e), "error");
    }
  }, [client, conversationId, draft, userId]);

  const retry = (message: PendingMessage) => {
    setPending((p) => p.filter((m) => m.id !== message.id));
    setDraft(message.body);
  };

  const leave = () => {
    void client.invalidateQueries({ queryKey: queryKeys.conversations });
    navigation.goBack();
  };

  const runConfirm = async () => {
    try {
      if (confirm === "unmatch" && conversation.data) {
        await discoveryApi.unmatch(conversation.data.matchId);
        showToast("Match supprimé", "success");
      } else if (confirm === "hide") {
        await chatApi.hide(conversationId);
        showToast("Conversation supprimée de votre liste", "success");
      }
      leave();
    } catch (e) {
      showToast(errorMessage(e), "error");
    }
  };

  if (messages.isLoading || conversation.isLoading) return <LoadingState />;
  if (conversation.error || messages.error) {
    return <ErrorState message={errorMessage(conversation.error ?? messages.error)} onRetry={() => void Promise.all([conversation.refetch(), messages.refetch()])} />;
  }

  return (
    <Screen
      edges={["bottom"]}
      footer={
        <View style={[styles.composer, { backgroundColor: colors.surface, borderColor: colors.border, borderRadius: radii.xl }]}>
          <TextInput
            accessibilityLabel="Votre message"
            maxLength={MAX_LENGTH}
            multiline
            onChangeText={setDraft}
            placeholder="Écrire un message…"
            placeholderTextColor={colors.textSubtle}
            style={[styles.input, { color: colors.text }]}
            testID="chat-input"
            value={draft}
          />
          <IconButton disabled={!draft.trim()} filled icon="arrow-up" label="Envoyer" onPress={() => void send()} size={40} testID="chat-send" tone="primary" />
        </View>
      }
      testID="chat-screen"
    >
      <FlatList
        ListEmptyComponent={
          <View style={styles.empty}>
            <Avatar name={other?.firstName ?? ""} ring size={88} uri={other?.photo?.url} />
            <Text align="center" variant="heading">
              Vous avez matché avec {other?.firstName}
            </Text>
            <Text align="center" tone="muted">
              Lancez la conversation : une question sur ses centres d’intérêt est un bon début.
            </Text>
          </View>
        }
        contentContainerStyle={styles.list}
        data={items}
        inverted={items.length > 0}
        keyExtractor={(m) => m.id}
        onEndReached={() => {
          if (messages.hasNextPage && !messages.isFetchingNextPage) void messages.fetchNextPage();
        }}
        renderItem={({ item, index }) => {
          const mine = item.senderId === userId;
          const isNewestMine = mine && items.findIndex((m) => m.senderId === userId) === index;
          const read = mine && !item.pending && otherLastReadAt !== null && new Date(item.createdAt) <= new Date(otherLastReadAt);
          return (
            <View style={[styles.bubbleRow, mine ? styles.mine : styles.theirs]}>
              <View
                style={[
                  styles.bubble,
                  {
                    backgroundColor: mine ? colors.primary : colors.surfaceMuted,
                    borderRadius: radii.lg,
                    opacity: item.pending ? 0.6 : 1
                  }
                ]}
                testID={mine ? "message-mine" : "message-theirs"}
              >
                <Text style={{ color: mine ? colors.onPrimary : colors.text }}>{item.body}</Text>
              </View>
              <Text onPress={item.failed ? () => retry(item) : undefined} tone={item.failed ? "danger" : "subtle"} variant="caption">
                {item.failed ? "Échec de l'envoi · Réessayer" : item.pending ? "Envoi…" : formatMessageTime(item.createdAt)}
                {isNewestMine && read ? " · Lu" : ""}
              </Text>
            </View>
          );
        }}
      />
      <ActionSheet
        actions={[
          ...(other ? [{ label: `Voir le profil de ${other.firstName}`, onPress: () => navigation.navigate("ProfileDetail", { userId: other.userId }) }] : []),
          { label: "Supprimer la conversation de ma liste", onPress: () => setConfirm("hide") },
          { label: "Annuler le match", destructive: true, onPress: () => setConfirm("unmatch"), testID: "chat-unmatch" },
          { label: "Signaler ou bloquer", destructive: true, onPress: () => setSafetyOpen(true), testID: "chat-safety" }
        ]}
        onClose={() => setMenuOpen(false)}
        visible={menuOpen}
      />
      <ActionSheet
        actions={[{ label: confirm === "unmatch" ? "Annuler le match" : "Supprimer", destructive: true, onPress: () => void runConfirm(), testID: "chat-confirm" }]}
        message={
          confirm === "unmatch"
            ? "La conversation sera supprimée pour vous deux et ce profil ne vous sera plus proposé."
            : "Elle disparaîtra de votre liste jusqu'au prochain message. Les anciens messages resteront masqués pour vous."
        }
        onClose={() => setConfirm(null)}
        title={confirm === "unmatch" ? "Annuler le match ?" : "Supprimer la conversation ?"}
        visible={confirm !== null}
      />
      {other ? <SafetySheet name={other.firstName} onBlocked={leave} onClose={() => setSafetyOpen(false)} userId={other.userId} visible={safetyOpen} /> : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  headerTitle: { flexDirection: "row", alignItems: "center", gap: 10 },
  list: { padding: 16, gap: 6, flexGrow: 1 },
  empty: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12, padding: 24 },
  bubbleRow: { maxWidth: "82%", gap: 2, marginVertical: 2 },
  mine: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { paddingHorizontal: 14, paddingVertical: 10 },
  composer: { flexDirection: "row", alignItems: "flex-end", borderWidth: 1, paddingLeft: 16, paddingRight: 6, paddingVertical: 6, gap: 8 },
  input: { flex: 1, fontSize: 16, maxHeight: 120, paddingVertical: 8 }
});
