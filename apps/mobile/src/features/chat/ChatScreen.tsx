import React, { useEffect, useMemo, useRef, useState } from "react";
import {
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  TextInput,
  View
} from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useMutation, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { SafeAreaView } from "react-native-safe-area-context";
import { ApiError } from "../../api/client";
import { chatApi, matchesApi } from "../../api/endpoints";
import type { Message, MessagesPage } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";
import { keys, useMessages } from "../../hooks/queries";
import { formatDayHeader, formatMessageTime } from "../../lib/format";
import { isRead, mergeMessage } from "../../lib/messages";
import type { RootParamList } from "../../navigation/types";
import { fontSizes, radii, spacing, useTheme } from "../../theme/theme";
import { IconButton } from "../../ui/Button";
import { EmptyState, ErrorState, LoadingView } from "../../ui/Feedback";
import { ConfirmDialog } from "../../ui/Modal";
import { Avatar } from "../../ui/Photo";
import { Text } from "../../ui/Text";
import { useToast } from "../../ui/Toast";
import { useSafetyMenu } from "../safety/useSafetyMenu";

type Props = NativeStackScreenProps<RootParamList, "Chat">;

type Row =
  { kind: "message"; message: Message; showDay: boolean } | { kind: "day"; label: string; id: string };

function Bubble({ message, mine, read }: { message: Message; mine: boolean; read: boolean }) {
  const { colors } = useTheme();
  return (
    <View style={[styles.bubbleWrap, mine ? styles.mine : styles.theirs]}>
      <View style={[styles.bubble, { backgroundColor: mine ? colors.bubbleMine : colors.bubbleTheirs }]}>
        <Text style={{ color: mine ? colors.onBubbleMine : colors.text }}>{message.body}</Text>
      </View>
      <View style={styles.meta}>
        <Text variant="caption" tone="textSubtle">
          {formatMessageTime(message.createdAt)}
        </Text>
        {mine ? (
          <Ionicons
            name={read ? "checkmark-done" : "checkmark"}
            size={14}
            color={read ? colors.primary : colors.textSubtle}
            accessibilityLabel={read ? "Lu" : "Envoyé"}
          />
        ) : null}
      </View>
    </View>
  );
}

export function ChatScreen({ navigation, route }: Props) {
  const { conversationId, matchId, peer } = route.params;
  const { user } = useAuth();
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const messagesQuery = useMessages(conversationId);
  const [text, setText] = useState("");
  const [confirm, setConfirm] = useState<"clear" | "unmatch" | null>(null);
  const listRef = useRef<FlatList<Row>>(null);
  const meId = user?.id ?? "";

  const markRead = useMutation({
    mutationFn: () => chatApi.markRead(conversationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      void queryClient.invalidateQueries({ queryKey: keys.unread });
      void queryClient.invalidateQueries({ queryKey: keys.notifications });
    }
  });

  const messages = useMemo(
    () => messagesQuery.data?.pages.flatMap((p) => p.messages) ?? [],
    [messagesQuery.data]
  );
  const peerLastReadAt = messagesQuery.data?.pages[0]?.peerLastReadAt ?? "1970-01-01T00:00:00Z";
  const newestId = messages[0]?.id;
  const newestFromPeer = messages[0] ? messages[0].senderId !== meId : false;

  // Mark as read when opening and whenever a new message from the peer arrives while the chat is open.
  const { mutate: markReadNow } = markRead;
  useEffect(() => {
    if (messagesQuery.isSuccess && (newestFromPeer || newestId === undefined)) markReadNow();
  }, [messagesQuery.isSuccess, newestId, newestFromPeer, markReadNow]);

  const rows = useMemo<Row[]>(() => {
    // The list is inverted (newest first). A day header goes *after* the oldest message of that day.
    const out: Row[] = [];
    messages.forEach((m, i) => {
      out.push({ kind: "message", message: m, showDay: false });
      const older = messages[i + 1];
      if (!older || new Date(older.createdAt).toDateString() !== new Date(m.createdAt).toDateString()) {
        out.push({ kind: "day", label: formatDayHeader(m.createdAt), id: `day-${m.id}` });
      }
    });
    return out;
  }, [messages]);

  const send = useMutation({
    mutationFn: (body: string) => chatApi.send(conversationId, body),
    onSuccess: (message) => {
      queryClient.setQueryData<InfiniteData<MessagesPage, string | undefined>>(
        keys.messages(conversationId),
        (data) => {
          if (!data || data.pages.length === 0) return data;
          const [first, ...rest] = data.pages;
          return { ...data, pages: [{ ...first, messages: mergeMessage(first.messages, message) }, ...rest] };
        }
      );
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      void queryClient.invalidateQueries({ queryKey: keys.matches });
    },
    onError: (e, body) => {
      setText((current) => current || body);
      toast.show(e instanceof ApiError ? e.message : "Message non envoyé.", "error");
    }
  });

  const submit = () => {
    const body = text.trim();
    if (!body || send.isPending) return;
    setText("");
    send.mutate(body);
  };

  const leave = () => navigation.popToTop();
  const clear = useMutation({
    mutationFn: () => chatApi.clear(conversationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      leave();
    },
    onError: (e) => toast.show(e instanceof Error ? e.message : "Action impossible.", "error")
  });
  const unmatch = useMutation({
    mutationFn: () => matchesApi.unmatch(matchId),
    onSuccess: () => {
      void queryClient.invalidateQueries();
      leave();
    },
    onError: (e) => toast.show(e instanceof Error ? e.message : "Action impossible.", "error")
  });

  const safety = useSafetyMenu({
    userId: peer.userId,
    firstName: peer.firstName,
    onBlocked: leave,
    extraActions: [
      {
        label: "Voir le profil",
        onPress: () => navigation.navigate("UserProfile", { userId: peer.userId, card: peer })
      },
      { label: "Supprimer la conversation", onPress: () => setConfirm("clear") },
      { label: "Se désapparier", destructive: true, onPress: () => setConfirm("unmatch") }
    ]
  });

  return (
    <SafeAreaView edges={["bottom"]} style={[styles.flex, { backgroundColor: colors.background }]}>
      <View style={[styles.header, { borderBottomColor: colors.border }]}>
        <IconButton icon="chevron-back" label="Retour" onPress={() => navigation.goBack()} />
        <Pressable
          style={styles.peer}
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${peer.firstName}`}
          onPress={() => navigation.navigate("UserProfile", { userId: peer.userId, card: peer })}
        >
          <Avatar path={peer.photos[0]?.url} name={peer.firstName} size={36} />
          <Text variant="heading" numberOfLines={1}>
            {peer.firstName}
          </Text>
        </Pressable>
        <IconButton
          icon="ellipsis-horizontal"
          label="Plus d'options"
          onPress={safety.open}
          testID="chat-menu"
        />
      </View>

      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        {messagesQuery.isPending ? (
          <LoadingView />
        ) : messagesQuery.isError ? (
          <ErrorState message={messagesQuery.error.message} onRetry={() => void messagesQuery.refetch()} />
        ) : (
          <FlatList
            ref={listRef}
            inverted
            data={rows}
            keyExtractor={(r) => (r.kind === "day" ? r.id : r.message.id)}
            contentContainerStyle={styles.list}
            renderItem={({ item }) =>
              item.kind === "day" ? (
                <Text variant="caption" tone="textSubtle" align="center" style={styles.day}>
                  {item.label}
                </Text>
              ) : (
                <Bubble
                  message={item.message}
                  mine={item.message.senderId === meId}
                  read={isRead(item.message, meId, peerLastReadAt)}
                />
              )
            }
            ListEmptyComponent={
              <View style={styles.invertedFix}>
                <EmptyState
                  icon="hand-left-outline"
                  title={`Dis bonjour à ${peer.firstName}`}
                  message="Un message simple et sincère est le meilleur début."
                />
              </View>
            }
            onEndReached={() => {
              if (messagesQuery.hasNextPage && !messagesQuery.isFetchingNextPage)
                void messagesQuery.fetchNextPage();
            }}
            onEndReachedThreshold={0.3}
            keyboardShouldPersistTaps="handled"
          />
        )}

        <View style={[styles.composer, { borderTopColor: colors.border, backgroundColor: colors.surface }]}>
          <TextInput
            value={text}
            onChangeText={setText}
            placeholder="Écris un message…"
            placeholderTextColor={colors.textSubtle}
            accessibilityLabel="Message"
            multiline
            maxLength={2000}
            style={[
              styles.input,
              { color: colors.text, backgroundColor: colors.background, borderColor: colors.border }
            ]}
            onSubmitEditing={Platform.OS === "web" ? submit : undefined}
            blurOnSubmit={false}
            testID="chat-input"
          />
          <IconButton
            icon="send"
            label="Envoyer"
            size={46}
            color={colors.onPrimary}
            background={colors.primary}
            disabled={!text.trim() || send.isPending}
            onPress={submit}
            testID="chat-send"
          />
        </View>
      </KeyboardAvoidingView>

      {safety.element}
      <ConfirmDialog
        visible={confirm === "clear"}
        title="Supprimer la conversation ?"
        message="L'historique disparaît de ton côté uniquement. Il reste visible par l'autre personne."
        confirmLabel="Supprimer"
        destructive
        loading={clear.isPending}
        onConfirm={() => clear.mutate()}
        onCancel={() => setConfirm(null)}
      />
      <ConfirmDialog
        visible={confirm === "unmatch"}
        title={`Se désapparier de ${peer.firstName} ?`}
        message="Le match et la conversation seront supprimés pour vous deux. Cette action est définitive."
        confirmLabel="Se désapparier"
        destructive
        loading={unmatch.isPending}
        onConfirm={() => unmatch.mutate()}
        onCancel={() => setConfirm(null)}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: spacing.xs,
    paddingVertical: spacing.xs,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  peer: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.xs },
  list: { paddingHorizontal: spacing.md, paddingVertical: spacing.sm },
  day: { marginVertical: spacing.sm },
  invertedFix: { transform: [{ scaleY: -1 }], flex: 1, minHeight: 300 },
  bubbleWrap: { marginVertical: 3, maxWidth: "82%" },
  mine: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirs: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { borderRadius: radii.lg, paddingHorizontal: spacing.md, paddingVertical: spacing.xs + 2 },
  meta: { flexDirection: "row", alignItems: "center", gap: 4, marginTop: 2 },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.xs,
    padding: spacing.xs,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    minHeight: 46,
    maxHeight: 120,
    borderWidth: 1,
    borderRadius: radii.xl,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 2,
    fontSize: fontSizes.md
  }
});
