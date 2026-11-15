import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FlatList, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useFocusEffect } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { chatApi } from "../../api/chat";
import { matchingApi } from "../../api/matching";
import { queryKeys } from "../../api/queryKeys";
import type { Message, MessagesPage } from "../../api/types";
import { errorMessage } from "../../lib/errors";
import { formatDayHeader, formatTime } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { useRealtimeConnected, useRealtimeEvent } from "../../realtime/RealtimeProvider";
import { useSession } from "../../session/SessionProvider";
import { radii, spacing, type, useTheme } from "../../theme";
import { ActionSheet, Avatar, ConfirmSheet, ErrorState, Header, Icon, IconButton, Loading, Screen, Text, toast } from "../../ui";
import { BlockConfirm, ReportSheet } from "../safety/SafetyActions";

type Props = NativeStackScreenProps<AppStackParamList, "Chat">;

type Row = { kind: "message"; message: Message; pending?: boolean; failed?: boolean } | { kind: "day"; key: string; label: string };

const MAX_LENGTH = 2000;

export function ChatScreen({ route, navigation }: Props) {
  const { conversationId } = route.params;
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const { user } = useSession();
  const queryClient = useQueryClient();
  const connected = useRealtimeConnected();
  const [draft, setDraft] = useState("");
  const [pending, setPending] = useState<{ id: string; body: string; createdAt: string; failed?: boolean }[]>([]);
  const [menu, setMenu] = useState(false);
  const [confirm, setConfirm] = useState<"clear" | "unmatch" | null>(null);
  const [reporting, setReporting] = useState(false);
  const [blocking, setBlocking] = useState(false);
  const inputRef = useRef<TextInput>(null);
  // Several paths can end this screen at once (our own unmatch/block/report
  // and the matching realtime "match.removed" event): leave exactly once.
  const left = useRef(false);
  const leave = useCallback(() => {
    if (left.current) return;
    left.current = true;
    navigation.goBack();
  }, [navigation]);

  const conversation = useQuery({ queryKey: queryKeys.conversation(conversationId), queryFn: () => chatApi.conversation(conversationId) });
  const messages = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    refetchInterval: connected ? false : 5_000
  });

  const markRead = useMutation({
    mutationFn: () => chatApi.markRead(conversationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.badges });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
    }
  });
  const markReadRef = useRef(markRead.mutate);
  markReadRef.current = markRead.mutate;

  useFocusEffect(
    useCallback(() => {
      markReadRef.current();
    }, [])
  );

  useRealtimeEvent((event) => {
    if (event.type === "message.new" && event.data.conversationId === conversationId && event.data.senderId !== user?.id) {
      markReadRef.current();
    }
    if (event.type === "match.removed" && conversation.data && event.data.matchId === conversation.data.matchId && !left.current) {
      toast("Cette conversation n'est plus disponible.");
      leave();
    }
  });

  const insertMessage = (msg: Message) =>
    queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conversationId), (data) => {
      if (!data || data.pages.length === 0) return data;
      if (data.pages.some((p) => p.messages.some((m) => m.id === msg.id))) return data;
      const [first, ...rest] = data.pages;
      return { ...data, pages: [{ ...first, messages: [msg, ...first.messages] }, ...rest] };
    });

  const send = useMutation({
    mutationFn: (v: { tempId: string; body: string }) => chatApi.send(conversationId, v.body),
    onSuccess: (msg, v) => {
      insertMessage(msg);
      setPending((p) => p.filter((x) => x.id !== v.tempId));
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    },
    onError: (e, v) => {
      setPending((p) => p.map((x) => (x.id === v.tempId ? { ...x, failed: true } : x)));
      toast(errorMessage(e), "error");
    }
  });

  const submit = () => {
    const body = draft.trim();
    if (!body) return;
    const tempId = `local-${Date.now()}`;
    setPending((p) => [{ id: tempId, body, createdAt: new Date().toISOString() }, ...p]);
    setDraft("");
    send.mutate({ tempId, body });
  };

  const retry = (id: string) => {
    const item = pending.find((p) => p.id === id);
    if (!item) return;
    setPending((p) => p.map((x) => (x.id === id ? { ...x, failed: false } : x)));
    send.mutate({ tempId: id, body: item.body });
  };

  const otherLastReadAt = messages.data?.pages[0]?.otherLastReadAt ?? conversation.data?.otherLastReadAt ?? null;
  const loaded = useMemo(() => messages.data?.pages.flatMap((p) => p.messages) ?? [], [messages.data]);

  const rows = useMemo<Row[]>(() => {
    const merged: Row[] = [
      ...pending.map((p) => ({
        kind: "message" as const,
        pending: !p.failed,
        failed: p.failed,
        message: { id: p.id, body: p.body, createdAt: p.createdAt, senderId: user?.id ?? "", conversationId }
      })),
      ...loaded.map((m) => ({ kind: "message" as const, message: m }))
    ];
    // Inverted list: insert a day header after the oldest message of each day.
    const out: Row[] = [];
    merged.forEach((row, i) => {
      out.push(row);
      const next = merged[i + 1];
      if (row.kind !== "message") return;
      const day = new Date(row.message.createdAt).toDateString();
      const nextDay = next && next.kind === "message" ? new Date(next.message.createdAt).toDateString() : null;
      if (day !== nextDay) out.push({ kind: "day", key: `day-${day}`, label: formatDayHeader(row.message.createdAt) });
    });
    return out;
  }, [pending, loaded, user?.id, conversationId]);

  const lastOwnId = loaded.find((m) => m.senderId === user?.id)?.id;

  useEffect(() => {
    if (conversation.isError) {
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    }
  }, [conversation.isError, queryClient]);

  const unmatch = useMutation({
    mutationFn: () => matchingApi.unmatch(conversation.data?.matchId ?? ""),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      toast("Match annulé.");
      leave();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });
  const clear = useMutation({
    mutationFn: () => chatApi.clear(conversationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.messages(conversationId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      setConfirm(null);
      toast("Conversation effacée de votre côté.");
      leave();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  if (conversation.isPending) return <Loading />;
  if (conversation.isError || !conversation.data) {
    return (
      <Screen>
        <Header onBack={() => navigation.goBack()} />
        <ErrorState message={errorMessage(conversation.error)} />
      </Screen>
    );
  }
  const other = conversation.data.user;

  const renderRow = ({ item }: { item: Row }) => {
    if (item.kind === "day") {
      return (
        <Text variant="caption" color="textSubtle" align="center" style={styles.day}>
          {item.label}
        </Text>
      );
    }
    const mine = item.message.senderId === user?.id;
    const isLastOwn = item.message.id === lastOwnId;
    const read = isLastOwn && otherLastReadAt !== null && Date.parse(otherLastReadAt) >= Date.parse(item.message.createdAt);
    return (
      <View style={[styles.bubbleRow, mine ? styles.right : styles.left]}>
        <Pressable
          disabled={!item.failed}
          onPress={() => retry(item.message.id)}
          style={[
            styles.bubble,
            mine
              ? { backgroundColor: colors.primary, borderBottomRightRadius: 6 }
              : { backgroundColor: colors.surfaceAlt, borderBottomLeftRadius: 6 },
            item.pending && { opacity: 0.6 },
            item.failed && { borderWidth: 1, borderColor: colors.danger }
          ]}
          accessibilityLabel={`${mine ? "Vous" : other.firstName}, ${formatTime(item.message.createdAt)} : ${item.message.body}`}
        >
          <Text color={mine ? "onPrimary" : "text"} selectable>
            {item.message.body}
          </Text>
        </Pressable>
        <Text variant="caption" color={item.failed ? "danger" : "textSubtle"} style={styles.meta}>
          {item.failed ? "Échec de l'envoi — touchez pour réessayer" : item.pending ? "Envoi…" : formatTime(item.message.createdAt)}
          {isLastOwn && !item.pending ? (read ? " · Lu" : " · Envoyé") : ""}
        </Text>
      </View>
    );
  };

  return (
    <Screen padded={false} edges={["top"]} testID="chat-screen">
      <View style={[styles.header, { borderBottomColor: colors.border }]}>
        <Header
          onBack={() => navigation.goBack()}
          title={other.firstName}
          subtitle={other.age ? `${other.age} ans${other.city ? ` · ${other.city}` : ""}` : undefined}
          right={
            <>
              <Pressable
                onPress={() => navigation.navigate("ProfileDetail", { userId: other.userId, source: "chat" })}
                accessibilityRole="button"
                accessibilityLabel={`Voir le profil de ${other.firstName}`}
              >
                <Avatar photo={other.photos[0]} name={other.firstName} size={40} />
              </Pressable>
              <IconButton
                icon="ellipsis-vertical"
                label="Options de la conversation"
                onPress={() => setMenu(true)}
                size={40}
                background="surface"
                testID="chat-menu"
              />
            </>
          }
        />
      </View>
      {messages.isPending ? (
        <Loading />
      ) : (
        <FlatList
          inverted
          data={rows}
          keyExtractor={(row) => (row.kind === "day" ? row.key : row.message.id)}
          renderItem={renderRow}
          contentContainerStyle={styles.list}
          onEndReached={() => messages.hasNextPage && !messages.isFetchingNextPage && void messages.fetchNextPage()}
          onEndReachedThreshold={0.3}
          keyboardShouldPersistTaps="handled"
          ListEmptyComponent={
            <View style={[styles.empty, { transform: [{ scaleY: -1 }] }]}>
              <Avatar photo={other.photos[0]} name={other.firstName} size={96} ring />
              <Text variant="heading" align="center">
                Vous avez matché avec {other.firstName}
              </Text>
              <Text color="textMuted" align="center">
                Un message sincère vaut mieux qu’un « salut ». Parlez-lui de{" "}
                {other.interests[0]?.label.toLowerCase() ?? "ce qui vous a plu"} !
              </Text>
            </View>
          }
        />
      )}
      <View
        style={[
          styles.composer,
          { borderTopColor: colors.border, paddingBottom: insets.bottom + spacing.xs, backgroundColor: colors.background }
        ]}
      >
        <TextInput
          ref={inputRef}
          value={draft}
          onChangeText={(v) => setDraft(v.slice(0, MAX_LENGTH))}
          placeholder="Votre message…"
          placeholderTextColor={colors.textSubtle}
          multiline
          style={[type.body, styles.input, { backgroundColor: colors.surfaceAlt, color: colors.text, borderColor: colors.border }]}
          accessibilityLabel="Votre message"
          testID="chat-input"
          onKeyPress={(e) => {
            // Enter sends on web; Shift+Enter keeps a new line.
            const native = e.nativeEvent as { key: string; shiftKey?: boolean };
            if (native.key === "Enter" && !native.shiftKey && typeof document !== "undefined") {
              (e as unknown as { preventDefault?: () => void }).preventDefault?.();
              submit();
            }
          }}
        />
        <Pressable
          onPress={submit}
          disabled={!draft.trim()}
          accessibilityRole="button"
          accessibilityLabel="Envoyer"
          testID="chat-send"
          style={[styles.send, { backgroundColor: draft.trim() ? colors.primary : colors.surfaceAlt }]}
        >
          <Icon name="arrow-up" size={22} color={draft.trim() ? "onPrimary" : "textSubtle"} />
        </Pressable>
      </View>
      <ActionSheet
        visible={menu}
        onClose={() => setMenu(false)}
        actions={[
          { label: "Voir le profil", onPress: () => navigation.navigate("ProfileDetail", { userId: other.userId, source: "chat" }) },
          { label: "Effacer la conversation", onPress: () => setConfirm("clear"), testID: "chat-clear" },
          { label: "Annuler le match", onPress: () => setConfirm("unmatch"), destructive: true, testID: "chat-unmatch" },
          { label: "Bloquer", onPress: () => setBlocking(true), destructive: true, testID: "chat-block" },
          { label: "Signaler", onPress: () => setReporting(true), destructive: true, testID: "chat-report" }
        ]}
      />
      <ConfirmSheet
        visible={confirm === "clear"}
        title="Effacer la conversation ?"
        message={`Les messages disparaîtront de votre appareil uniquement. ${other.firstName} les conserve.`}
        confirmLabel="Effacer"
        destructive
        loading={clear.isPending}
        onConfirm={() => clear.mutate()}
        onClose={() => setConfirm(null)}
      />
      <ConfirmSheet
        visible={confirm === "unmatch"}
        title={`Annuler le match avec ${other.firstName} ?`}
        message="La conversation sera supprimée pour vous deux et vous ne vous recroiserez plus dans la découverte."
        confirmLabel="Annuler le match"
        destructive
        loading={unmatch.isPending}
        onConfirm={() => unmatch.mutate()}
        onClose={() => setConfirm(null)}
      />
      <ReportSheet
        visible={reporting}
        userId={other.userId}
        name={other.firstName}
        onClose={() => setReporting(false)}
        onDone={() => {
          setReporting(false);
          leave();
        }}
      />
      <BlockConfirm
        visible={blocking}
        userId={other.userId}
        name={other.firstName}
        onClose={() => setBlocking(false)}
        onDone={() => {
          setBlocking(false);
          leave();
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { paddingHorizontal: spacing.md, borderBottomWidth: StyleSheet.hairlineWidth },
  list: { padding: spacing.md, gap: 4, flexGrow: 1 },
  day: { marginVertical: spacing.md },
  bubbleRow: { maxWidth: "82%", marginVertical: 2 },
  left: { alignSelf: "flex-start" },
  right: { alignSelf: "flex-end", alignItems: "flex-end" },
  bubble: { paddingHorizontal: spacing.md, paddingVertical: 10, borderRadius: radii.lg },
  meta: { marginTop: 3, marginHorizontal: 6, fontSize: 11 },
  empty: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.sm, padding: spacing.xl },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    paddingTop: spacing.xs,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    maxHeight: 140,
    minHeight: 44,
    borderRadius: 22,
    borderWidth: 1,
    paddingHorizontal: spacing.md,
    paddingTop: 10,
    paddingBottom: 10,
    outlineStyle: "none"
  } as object,
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" }
});
