import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import {
  Alert,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  TextInput,
  View
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { blockUser, getConversation, markRead, sendMessage } from "../../api/social";
import type { Message, MessagesPage } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { useMessages } from "../../hooks/queries";
import { formatDayHeader, formatTime } from "../../lib/dates";
import { errorMessage } from "../../lib/errors";
import { insertMessage } from "../../realtime/RealtimeProvider";
import type { MainStackParams } from "../../navigation/types";
import { ErrorState, Loading, Text, useActionSheet, useToast } from "../../ui";
import { MIN_TOUCH, radius, spacing } from "../../theme/tokens";
import { useTheme } from "../../theme/ThemeProvider";

type Props = NativeStackScreenProps<MainStackParams, "Chat">;

const MAX_LENGTH = 2000;

export default function ChatScreen({ navigation, route }: Props) {
  const { conversationId, name } = route.params;
  const { colors } = useTheme();
  const { user } = useAuth();
  const toast = useToast();
  const sheet = useActionSheet();
  const queryClient = useQueryClient();
  const messages = useMessages(conversationId);
  const conversation = useQuery({
    queryKey: ["conversation", conversationId],
    queryFn: () => getConversation(conversationId)
  });
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const inputRef = useRef<TextInput>(null);

  const peer = conversation.data?.profile;

  const items = useMemo(() => messages.data?.pages.flatMap((p) => p.messages) ?? [], [messages.data]);
  const peerReadAt = messages.data?.pages[0]?.peerLastReadAt;
  const newestId = items[0]?.id;
  const newestFromPeer = items[0] ? items[0].senderId !== user?.id : false;

  // Mark as read on open and whenever a message from the other person lands while open.
  useEffect(() => {
    if (!messages.isSuccess) return;
    if (items.length === 0 || newestFromPeer) {
      markRead(conversationId)
        .then(() => queryClient.invalidateQueries({ queryKey: ["conversations"] }))
        .catch(() => undefined);
      void queryClient.invalidateQueries({ queryKey: ["notifications"] });
    }
  }, [conversationId, messages.isSuccess, newestId, newestFromPeer, items.length, queryClient]);

  const openMenu = useCallback(() => {
    if (!peer) return;
    sheet.show({
      title: peer.firstName,
      actions: [
        {
          label: "Voir le profil",
          onPress: () =>
            navigation.navigate("ProfileDetail", { userId: peer.userId, profile: peer, conversationId })
        },
        {
          label: "Signaler",
          onPress: () => navigation.navigate("Report", { userId: peer.userId, name: peer.firstName })
        },
        {
          label: "Bloquer",
          destructive: true,
          onPress: () =>
            Alert.alert(
              `Bloquer ${peer.firstName} ?`,
              "La conversation et le match seront supprimés. Cette personne ne pourra plus vous voir.",
              [
                { text: "Annuler", style: "cancel" },
                {
                  text: "Bloquer",
                  style: "destructive",
                  onPress: () => {
                    blockUser(peer.userId)
                      .then(() => {
                        void queryClient.invalidateQueries();
                        navigation.popToTop();
                        toast.show(`${peer.firstName} a été bloqué(e).`, "success");
                      })
                      .catch((e) => toast.show(errorMessage(e), "error"));
                  }
                }
              ]
            )
        }
      ]
    });
  }, [conversationId, navigation, peer, queryClient, sheet, toast]);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: name ?? peer?.firstName ?? "Conversation",
      headerRight: () => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Options de la conversation"
          onPress={openMenu}
          style={styles.menu}
          testID="chat-menu"
        >
          <Text variant="title">⋯</Text>
        </Pressable>
      )
    });
  }, [navigation, name, peer?.firstName, openMenu]);

  const send = async () => {
    const body = draft.trim();
    if (!body || sending) return;
    setSending(true);
    try {
      const message = await sendMessage(conversationId, body);
      queryClient.setQueryData<InfiniteData<MessagesPage>>(["messages", conversationId], (cache) =>
        insertMessage(cache, message)
      );
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
      setDraft("");
    } catch (e) {
      toast.show(errorMessage(e), "error");
    } finally {
      setSending(false);
    }
  };

  if (messages.isPending) return <Loading />;
  if (messages.isError) return <ErrorState error={messages.error} onRetry={() => void messages.refetch()} />;

  return (
    <SafeAreaView
      edges={["bottom"]}
      style={[styles.flex, { backgroundColor: colors.background }]}
      testID="chat-screen"
    >
      <KeyboardAvoidingView
        style={styles.flex}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}
      >
        <FlatList
          testID="messages-list"
          inverted
          data={items}
          keyExtractor={(m) => m.id}
          contentContainerStyle={styles.list}
          onEndReached={() => messages.hasNextPage && void messages.fetchNextPage()}
          onEndReachedThreshold={0.4}
          ListEmptyComponent={
            <Text tone="muted" style={styles.empty}>
              Vous avez matché ! Envoyez le premier message à {name ?? "votre match"}.
            </Text>
          }
          renderItem={({ item, index }) => (
            <Bubble
              message={item}
              mine={item.senderId === user?.id}
              older={items[index + 1]}
              read={!!peerReadAt && Date.parse(item.createdAt) <= Date.parse(peerReadAt)}
            />
          )}
        />
        <View style={[styles.composer, { borderColor: colors.border, backgroundColor: colors.surface }]}>
          <TextInput
            ref={inputRef}
            testID="chat-input"
            accessibilityLabel="Votre message"
            value={draft}
            onChangeText={setDraft}
            placeholder="Écrire un message…"
            placeholderTextColor={colors.textMuted}
            multiline
            maxLength={MAX_LENGTH}
            style={[styles.input, { color: colors.text, backgroundColor: colors.surfaceAlt }]}
          />
          <Pressable
            testID="chat-send"
            accessibilityRole="button"
            accessibilityLabel="Envoyer"
            accessibilityState={{ disabled: !draft.trim() || sending }}
            disabled={!draft.trim() || sending}
            onPress={() => void send()}
            style={[
              styles.send,
              { backgroundColor: colors.accent, opacity: !draft.trim() || sending ? 0.45 : 1 }
            ]}
          >
            <Text tone="onAccent" bold>
              ➤
            </Text>
          </Pressable>
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

function Bubble({
  message,
  mine,
  older,
  read
}: {
  message: Message;
  mine: boolean;
  older?: Message;
  read: boolean;
}) {
  const { colors } = useTheme();
  const newDay =
    !older || new Date(older.createdAt).toDateString() !== new Date(message.createdAt).toDateString();
  return (
    <View>
      {newDay ? (
        <Text variant="caption" tone="muted" style={styles.day}>
          {formatDayHeader(message.createdAt)}
        </Text>
      ) : null}
      <View style={[styles.bubbleRow, mine ? styles.mine : styles.theirs]}>
        <View
          style={[
            styles.bubble,
            { backgroundColor: mine ? colors.accent : colors.surface, borderColor: colors.border },
            mine ? null : styles.bordered
          ]}
        >
          <Text tone={mine ? "onAccent" : "default"} selectable>
            {message.body}
          </Text>
          <Text
            variant="caption"
            tone={mine ? "onAccent" : "muted"}
            style={styles.time}
            accessibilityLabel={`${formatTime(message.createdAt)}${mine ? (read ? ", lu" : ", envoyé") : ""}`}
          >
            {formatTime(message.createdAt)}
            {mine ? (read ? "  ✓✓" : "  ✓") : ""}
          </Text>
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  list: { padding: spacing.md, gap: spacing.xs },
  empty: { textAlign: "center", padding: spacing.xl, transform: [{ scaleY: -1 }] },
  day: { textAlign: "center", marginVertical: spacing.md },
  bubbleRow: { flexDirection: "row", marginVertical: 2 },
  mine: { justifyContent: "flex-end" },
  theirs: { justifyContent: "flex-start" },
  bubble: {
    maxWidth: "80%",
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm
  },
  bordered: { borderWidth: 1 },
  time: { alignSelf: "flex-end", marginTop: 2, opacity: 0.85 },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: MIN_TOUCH,
    borderRadius: radius.lg,
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md,
    paddingBottom: spacing.md,
    fontSize: 16
  },
  send: {
    width: MIN_TOUCH,
    height: MIN_TOUCH,
    borderRadius: radius.pill,
    alignItems: "center",
    justifyContent: "center"
  },
  menu: { minWidth: MIN_TOUCH, minHeight: MIN_TOUCH, alignItems: "center", justifyContent: "center" }
});
