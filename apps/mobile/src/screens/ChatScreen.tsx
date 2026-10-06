import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { Alert, FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useFocusEffect } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { hideMatch, listMessages, markMatchRead, sendMessage, unmatch } from "../api/dating";
import { blockUser } from "../api/platform";
import type { ChatMessage } from "../api/types";
import { useAuth } from "../hooks/useAuth";
import { describeError } from "../lib/errors";
import { formatClock } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { useRealtimeEvents } from "../realtime/RealtimeProvider";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Avatar, MessageItem, Text, radii, spacing, typography, useTheme } from "../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "Chat">;

type Pending = { tempId: string; body: string; failed: boolean };

type Row = { kind: "message"; key: string; message: ChatMessage } | { kind: "pending"; key: string; pending: Pending };

function mergeMessages(current: ChatMessage[], incoming: ChatMessage[]): ChatMessage[] {
  const byId = new Map(current.map((message) => [message.id, message]));
  incoming.forEach((message) => byId.set(message.id, message));
  return [...byId.values()].sort((a, b) => Date.parse(a.createdAt) - Date.parse(b.createdAt) || a.id.localeCompare(b.id));
}

export default function ChatScreen({ navigation, route }: Props) {
  const { matchId, peerId, peerName } = route.params;
  const { colors } = useTheme();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const myId = user?.id ?? "";

  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [pending, setPending] = useState<Pending[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [peerReadAt, setPeerReadAt] = useState<string | null>(null);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [errorMessage, setErrorMessage] = useState("");
  const [loadingEarlier, setLoadingEarlier] = useState(false);
  const [draft, setDraft] = useState("");
  const focused = useRef(true);
  const stateRef = useRef(state);
  stateRef.current = state;

  const markRead = useCallback(() => {
    void markMatchRead(matchId)
      .then(() => queryClient.invalidateQueries({ queryKey: ["matches"] }))
      .then(() => queryClient.invalidateQueries({ queryKey: ["notifications"] }))
      .catch(() => undefined);
  }, [matchId, queryClient]);

  const load = useCallback(async () => {
    setState("loading");
    try {
      const page = await listMessages(matchId);
      setMessages((current) => mergeMessages(current, page.messages));
      setHasMore(page.hasMore);
      setPeerReadAt(page.peerLastReadAt);
      setState("ready");
      markRead();
    } catch (err) {
      const gone = (err as { status?: number }).status === 404;
      setErrorMessage(gone ? "Cette conversation n'existe plus." : describeError(err));
      setState("error");
    }
  }, [markRead, matchId]);

  useEffect(() => {
    void load();
  }, [load]);

  useFocusEffect(
    useCallback(() => {
      focused.current = true;
      if (stateRef.current === "ready") {
        markRead(); // messages may have arrived while another screen was on top
      }
      return () => {
        focused.current = false;
      };
    }, [markRead])
  );

  useRealtimeEvents((event) => {
    if (event.type === "message.new" && event.data.matchId === matchId) {
      setMessages((current) => mergeMessages(current, [event.data]));
      if (focused.current) {
        markRead();
      }
    } else if (event.type === "resync") {
      void load();
    } else if (event.type === "message.read" && event.data.matchId === matchId) {
      setPeerReadAt(event.data.readAt);
    } else if (event.type === "match.ended" && event.data.matchId === matchId) {
      showToast(`${peerName} a mis fin à la conversation.`);
      navigation.popToTop();
    }
  });

  const loadEarlier = async () => {
    const oldest = messages[0];
    if (!hasMore || loadingEarlier || !oldest) {
      return;
    }
    setLoadingEarlier(true);
    try {
      const page = await listMessages(matchId, oldest.id);
      setMessages((current) => mergeMessages(current, page.messages));
      setHasMore(page.hasMore);
    } catch (err) {
      showToast(describeError(err), "danger");
    } finally {
      setLoadingEarlier(false);
    }
  };

  const deliver = async (item: Pending) => {
    try {
      const sent = await sendMessage(matchId, item.body);
      setMessages((current) => mergeMessages(current, [sent]));
      setPending((current) => current.filter((p) => p.tempId !== item.tempId));
      void queryClient.invalidateQueries({ queryKey: ["matches"] });
    } catch (err) {
      const status = (err as { status?: number }).status;
      if (status === 404) {
        showToast("Cette conversation n'est plus disponible.", "danger");
        navigation.popToTop();
        return;
      }
      setPending((current) => current.map((p) => (p.tempId === item.tempId ? { ...p, failed: true } : p)));
      if (status !== undefined && status < 500 && status !== 429) {
        showToast(describeError(err, "Message non envoyé."), "danger");
      }
    }
  };

  const send = () => {
    const body = draft.trim();
    if (!body) {
      return;
    }
    const item: Pending = { tempId: `tmp-${Date.now()}-${Math.random()}`, body, failed: false };
    setDraft("");
    setPending((current) => [...current, item]);
    void deliver(item);
  };

  const retry = (item: Pending) => {
    setPending((current) => current.map((p) => (p.tempId === item.tempId ? { ...p, failed: false } : p)));
    void deliver(item);
  };

  const leave = (message: string) => {
    void queryClient.invalidateQueries({ queryKey: ["matches"] });
    showToast(message, "success");
    navigation.popToTop();
  };

  const menu = useCallback(() => {
    Alert.alert(peerName, undefined, [
      { text: "Voir le profil", onPress: () => navigation.navigate("ProfileDetail", { userId: peerId, context: "match", matchId }) },
      {
        text: "Masquer la conversation",
        onPress: () => {
          void hideMatch(matchId)
            .then(() => leave("Conversation masquée"))
            .catch((err) => showToast(describeError(err), "danger"));
        }
      },
      {
        text: "Défaire le match",
        style: "destructive",
        onPress: () =>
          Alert.alert("Défaire le match ?", "La conversation sera fermée pour vous deux.", [
            { text: "Annuler", style: "cancel" },
            {
              text: "Défaire",
              style: "destructive",
              onPress: () => {
                void unmatch(matchId)
                  .then(() => leave("Match défait"))
                  .catch((err) => showToast(describeError(err), "danger"));
              }
            }
          ])
      },
      { text: "Signaler", onPress: () => navigation.navigate("Report", { userId: peerId, name: peerName }) },
      {
        text: "Bloquer",
        style: "destructive",
        onPress: () =>
          Alert.alert(`Bloquer ${peerName} ?`, "Vous ne vous verrez plus et ne pourrez plus vous écrire.", [
            { text: "Annuler", style: "cancel" },
            {
              text: "Bloquer",
              style: "destructive",
              onPress: () => {
                void blockUser(peerId)
                  .then(() => leave("Personne bloquée"))
                  .catch((err) => showToast(describeError(err), "danger"));
              }
            }
          ])
      },
      { text: "Annuler", style: "cancel" }
    ]);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [matchId, navigation, peerId, peerName]);

  useLayoutEffect(() => {
    navigation.setOptions({
      headerTitle: () => (
        <Pressable
          accessibilityLabel={`Profil de ${peerName}`}
          accessibilityRole="button"
          onPress={() => navigation.navigate("ProfileDetail", { userId: peerId, context: "match", matchId })}
          style={styles.titleRow}
        >
          <Avatar name={peerName} size={32} />
          <Text weight="semibold">{peerName}</Text>
        </Pressable>
      ),
      headerRight: () => (
        <Pressable accessibilityLabel="Plus d'options" accessibilityRole="button" hitSlop={12} onPress={menu} testID="chat-menu">
          <Ionicons color={colors.text} name="ellipsis-horizontal" size={24} />
        </Pressable>
      )
    });
  }, [colors.text, matchId, menu, navigation, peerId, peerName]);

  const lastMineId = useMemo(() => [...messages].reverse().find((message) => message.senderId === myId)?.id, [messages, myId]);

  // Inverted list: newest first.
  const rows: Row[] = useMemo(
    () => [
      ...[...pending].reverse().map((item): Row => ({ kind: "pending", key: item.tempId, pending: item })),
      ...[...messages].reverse().map((message): Row => ({ kind: "message", key: message.id, message }))
    ],
    [messages, pending]
  );

  if (state === "loading" && messages.length === 0) {
    return <LoadingView fullScreen label="Chargement de la conversation…" />;
  }
  if (state === "error" && messages.length === 0) {
    return <ErrorView message={errorMessage} onAction={() => void load()} />;
  }

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : undefined}
      keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}
      style={styles.flex}
    >
      <FlatList
        contentContainerStyle={styles.list}
        data={rows}
        inverted
        keyExtractor={(row) => row.key}
        ListEmptyComponent={
          <View style={styles.empty}>
            <Text style={styles.center} tone="muted">
              C'est un match ! Brisez la glace en envoyant le premier message à {peerName}.
            </Text>
          </View>
        }
        ListFooterComponent={hasMore ? <LoadingView label={loadingEarlier ? "Chargement…" : undefined} style={styles.earlier} /> : null}
        onEndReached={() => void loadEarlier()}
        onEndReachedThreshold={0.4}
        renderItem={({ item }) => {
          if (item.kind === "pending") {
            return (
              <Pressable disabled={!item.pending.failed} onPress={() => retry(item.pending)}>
                <MessageItem
                  body={item.pending.body}
                  mine
                  status={item.pending.failed ? "Échec · toucher pour réessayer" : "Envoi…"}
                  time=""
                />
              </Pressable>
            );
          }
          const { message } = item;
          const mine = message.senderId === myId;
          const read = mine && message.id === lastMineId && peerReadAt !== null && Date.parse(message.createdAt) <= Date.parse(peerReadAt);
          return <MessageItem body={message.body} mine={mine} status={read ? "Lu" : undefined} time={formatClock(message.createdAt)} />;
        }}
        testID="chat-screen"
      />
      <View style={[styles.composer, { backgroundColor: colors.surface, borderTopColor: colors.border }]}>
        <TextInput
          accessibilityLabel="Votre message"
          maxLength={2000}
          multiline
          onChangeText={setDraft}
          placeholder="Écrire un message…"
          placeholderTextColor={colors.textSubtle}
          style={[typography.body, styles.input, { backgroundColor: colors.surfaceMuted, color: colors.text }]}
          testID="chat-message-input"
          value={draft}
        />
        <Pressable
          accessibilityLabel="Envoyer"
          accessibilityRole="button"
          disabled={draft.trim().length === 0}
          onPress={send}
          style={[styles.send, { backgroundColor: colors.primary, opacity: draft.trim().length === 0 ? 0.4 : 1 }]}
          testID="chat-send-button"
        >
          <Ionicons color={colors.primaryForeground} name="arrow-up" size={22} />
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  titleRow: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  list: { paddingVertical: spacing.md },
  empty: { padding: spacing.xxl, transform: [{ scaleY: -1 }] },
  center: { textAlign: "center" },
  earlier: { paddingVertical: spacing.md },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    borderRadius: radii.lg,
    paddingHorizontal: spacing.lg,
    paddingTop: 10,
    paddingBottom: 10
  },
  send: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" }
});
