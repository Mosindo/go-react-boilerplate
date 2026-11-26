import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ActivityIndicator, FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useFocusEffect } from "@react-navigation/native";
import { useQuery, type InfiniteData } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { getConversation, markConversationRead, sendMessage } from "../../api/chat";
import { errorMessage } from "../../api/client";
import { unmatch } from "../../api/matching";
import type { Message, MessagesPage } from "../../api/types";
import { SafetyDialogs } from "../../components/SafetyActions";
import { useMessages } from "../../hooks/queries";
import { queryClient, queryKeys } from "../../hooks/queryClient";
import { useAuth } from "../../hooks/useAuth";
import type { AppStackParamList } from "../../navigation/types";
import { layout, radii, spacing, typography, useTheme } from "../../theme";
import { ActionSheet, Avatar, ConfirmDialog, EmptyState, IconButton, LoadingState, Text, showToast } from "../../ui";
import { formatDayLabel, formatTime, sameDay } from "../../utils/format";

type Props = NativeStackScreenProps<AppStackParamList, "Chat">;

type PendingMessage = Message & { pending: true };
const MAX_LENGTH = 2000;

export function ChatScreen({ navigation, route }: Props) {
  const { conversationId } = route.params;
  const { colors } = useTheme();
  const { user } = useAuth();
  const meta = useQuery({ queryKey: queryKeys.conversation(conversationId), queryFn: () => getConversation(conversationId) });
  const messages = useMessages(conversationId);
  const [draft, setDraft] = useState("");
  const [pending, setPending] = useState<PendingMessage[]>([]);
  const [menu, setMenu] = useState(false);
  const [confirmUnmatch, setConfirmUnmatch] = useState(false);
  const [safety, setSafety] = useState<"block" | "report" | null>(null);
  const focused = useRef(false);

  const serverMessages = useMemo(() => messages.data?.pages.flatMap((p) => p.messages) ?? [], [messages.data]);
  const otherLastReadAt = messages.data?.pages[0]?.otherLastReadAt ?? meta.data?.otherLastReadAt ?? null;
  const all: (Message | PendingMessage)[] = useMemo(() => [...pending, ...serverMessages], [pending, serverMessages]);
  const lastMineId = useMemo(() => serverMessages.find((m) => m.senderId === user?.id)?.id, [serverMessages, user?.id]);
  const newestIncoming = serverMessages.find((m) => m.senderId !== user?.id)?.id;

  const markRead = useCallback(() => {
    markConversationRead(conversationId)
      .then(() => {
        void queryClient.invalidateQueries({ queryKey: queryKeys.unreadConversations });
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        void queryClient.invalidateQueries({ queryKey: queryKeys.unreadNotifications });
      })
      .catch(() => undefined);
  }, [conversationId]);

  useFocusEffect(
    useCallback(() => {
      focused.current = true;
      markRead();
      return () => {
        focused.current = false;
      };
    }, [markRead])
  );

  // A new incoming message while the thread is open is read immediately.
  useEffect(() => {
    if (newestIncoming && focused.current) markRead();
  }, [newestIncoming, markRead]);

  const send = async () => {
    const body = draft.trim();
    if (!body) return;
    const temp: PendingMessage = {
      id: `pending-${Date.now()}`,
      conversationId,
      senderId: user?.id ?? "",
      body,
      createdAt: new Date().toISOString(),
      pending: true
    };
    setDraft("");
    setPending((p) => [temp, ...p]);
    try {
      const saved = await sendMessage(conversationId, body);
      queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conversationId), (data) => {
        if (!data || data.pages.length === 0) return data;
        if (data.pages.some((p) => p.messages.some((m) => m.id === saved.id))) return data;
        const [first, ...rest] = data.pages;
        return { ...data, pages: [{ ...first, messages: [saved, ...first.messages] }, ...rest] };
      });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
    } catch (e) {
      setDraft(body);
      showToast(errorMessage(e, "Message non envoyé."), "error");
    } finally {
      setPending((p) => p.filter((m) => m.id !== temp.id));
    }
  };

  const doUnmatch = async () => {
    if (!meta.data) return;
    try {
      await unmatch(meta.data.matchId);
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      showToast("Match annulé.", "success");
      navigation.goBack();
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setConfirmUnmatch(false);
    }
  };

  const gone = meta.isError || messages.isError;
  const profile = meta.data?.profile;

  const renderItem = ({ item, index }: { item: Message | PendingMessage; index: number }) => {
    const mine = item.senderId === user?.id;
    const older = all[index + 1];
    const showDay = !older || !sameDay(older.createdAt, item.createdAt);
    const isPending = "pending" in item;
    const read = mine && item.id === lastMineId && otherLastReadAt !== null && Date.parse(otherLastReadAt) >= Date.parse(item.createdAt);
    return (
      <View>
        {showDay ? (
          <Text center style={styles.day} tone="subtle" variant="caption">
            {formatDayLabel(item.createdAt)}
          </Text>
        ) : null}
        <View style={[styles.bubbleRow, mine ? styles.mineRow : styles.theirRow]}>
          <View
            style={[
              styles.bubble,
              mine
                ? { backgroundColor: colors.primary, borderBottomRightRadius: 6 }
                : { backgroundColor: colors.surface, borderColor: colors.border, borderWidth: 1, borderBottomLeftRadius: 6 },
              isPending ? styles.pending : null
            ]}
            testID={mine ? "message-mine" : "message-theirs"}
          >
            <Text style={{ color: mine ? colors.onPrimary : colors.text }}>{item.body}</Text>
          </View>
          <Text style={styles.meta} tone="subtle" variant="caption">
            {isPending ? "Envoi…" : formatTime(item.createdAt)}
            {mine && item.id === lastMineId && !isPending ? (read ? " · Lu" : " · Envoyé") : ""}
          </Text>
        </View>
      </View>
    );
  };

  return (
    <SafeAreaView edges={["top", "bottom"]} style={[styles.root, { backgroundColor: colors.background }]} testID="chat-screen">
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.root}>
        <View style={[styles.header, { borderColor: colors.border }]}>
          <IconButton icon="chevron-back" label="Retour" onPress={navigation.goBack} testID="chat-back" />
          {profile ? (
            <Pressable
              accessibilityRole="button"
              onPress={() => navigation.navigate("ProfileDetail", { userId: profile.userId, source: "match" })}
              style={styles.headerProfile}
            >
              <Avatar name={profile.firstName} size={40} url={profile.photos[0]?.url} />
              <Text numberOfLines={1} variant="heading">
                {profile.firstName}
              </Text>
            </Pressable>
          ) : (
            <View style={styles.headerProfile} />
          )}
          {profile ? (
            <IconButton icon="ellipsis-horizontal" label="Options de la conversation" onPress={() => setMenu(true)} testID="chat-menu" />
          ) : null}
        </View>

        {gone ? (
          <EmptyState
            icon="chatbubble-outline"
            message="Ce match a pris fin ou la conversation n'est plus disponible."
            title="Conversation terminée"
          />
        ) : messages.isLoading ? (
          <LoadingState />
        ) : (
          <FlatList
            contentContainerStyle={styles.list}
            data={all}
            inverted
            keyExtractor={(m) => m.id}
            ListEmptyComponent={
              <View style={styles.emptyChat}>
                <Text center tone="muted">
                  C’est un match ! Dites bonjour à {profile?.firstName ?? "votre match"} 👋
                </Text>
              </View>
            }
            ListFooterComponent={messages.isFetchingNextPage ? <ActivityIndicator color={colors.primary} /> : null}
            onEndReached={() => {
              if (messages.hasNextPage && !messages.isFetchingNextPage) void messages.fetchNextPage();
            }}
            renderItem={renderItem}
            style={styles.flex}
          />
        )}

        {!gone ? (
          <View style={[styles.composer, { borderColor: colors.border, backgroundColor: colors.background }]}>
            <TextInput
              accessibilityLabel="Votre message"
              maxLength={MAX_LENGTH}
              multiline
              numberOfLines={Platform.OS === "web" ? 1 : undefined}
              onChangeText={setDraft}
              onKeyPress={(e) => {
                const ev = e.nativeEvent as { key: string; shiftKey?: boolean };
                if (Platform.OS === "web" && ev.key === "Enter" && !ev.shiftKey) {
                  (e as unknown as { preventDefault: () => void }).preventDefault();
                  void send();
                }
              }}
              placeholder="Écrire un message…"
              placeholderTextColor={colors.textSubtle}
              style={[typography.body, styles.input, { color: colors.text, backgroundColor: colors.surface, borderColor: colors.border }]}
              testID="chat-input"
              value={draft}
            />
            <IconButton
              disabled={!draft.trim()}
              icon="arrow-up"
              label="Envoyer"
              onPress={() => void send()}
              size={44}
              testID="chat-send"
              variant="primary"
            />
          </View>
        ) : null}
      </KeyboardAvoidingView>

      {profile ? (
        <>
          <ActionSheet
            actions={[
              {
                label: "Voir le profil",
                icon: "person-outline",
                onPress: () => navigation.navigate("ProfileDetail", { userId: profile.userId, source: "match" })
              },
              { label: "Annuler le match", icon: "heart-dislike-outline", onPress: () => setConfirmUnmatch(true), testID: "chat-unmatch" },
              { label: `Bloquer ${profile.firstName}`, icon: "hand-left-outline", onPress: () => setSafety("block") },
              { label: "Signaler", icon: "flag-outline", destructive: true, onPress: () => setSafety("report") }
            ]}
            onClose={() => setMenu(false)}
            title={profile.firstName}
            visible={menu}
          />
          <ConfirmDialog
            confirmLabel="Annuler le match"
            destructive
            message={`La conversation avec ${profile.firstName} sera supprimée pour vous deux.`}
            onCancel={() => setConfirmUnmatch(false)}
            onConfirm={doUnmatch}
            title="Annuler ce match ?"
            visible={confirmUnmatch}
          />
          <SafetyDialogs
            firstName={profile.firstName}
            mode={safety}
            onClose={() => setSafety(null)}
            onDone={() => {
              setSafety(null);
              navigation.goBack();
            }}
            userId={profile.userId}
          />
        </>
      ) : null}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  flex: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    paddingHorizontal: spacing.xs,
    paddingVertical: spacing.xs,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  headerProfile: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.sm },
  list: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.md,
    gap: 2,
    flexGrow: 1,
    width: "100%",
    maxWidth: layout.maxContentWidth,
    alignSelf: "center"
  },
  day: { marginVertical: spacing.sm, textTransform: "capitalize" },
  bubbleRow: { marginVertical: 3, maxWidth: "82%" },
  mineRow: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirRow: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { borderRadius: radii.lg, paddingHorizontal: spacing.md, paddingVertical: 10 },
  pending: { opacity: 0.6 },
  meta: { marginTop: 2, marginHorizontal: 4, fontSize: 11 },
  emptyChat: { flex: 1, justifyContent: "center", padding: spacing.xl, transform: [{ scaleY: -1 }] },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth,
    width: "100%",
    maxWidth: layout.maxContentWidth,
    alignSelf: "center"
  },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 140,
    borderWidth: 1,
    borderRadius: 22,
    paddingHorizontal: spacing.md,
    paddingTop: 10,
    paddingBottom: 10,
    outlineStyle: "none"
  } as object
});
