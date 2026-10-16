import React, { useCallback, useEffect, useMemo, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useIsFocused } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, absoluteUrl } from "../api/client";
import { friendlyError } from "../api/errors";
import { listMessages, markConversationRead, sendMessage } from "../api/platform";
import type { MessagesPage } from "../api/types";
import { ActionSheet, type SheetAction } from "../components/ActionSheet";
import { ReportSheet } from "../components/ReportSheet";
import { queryKeys } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { useRealtime } from "../hooks/useRealtime";
import { useSafetyActions } from "../hooks/useSafetyActions";
import { formatClock, formatDayLabel } from "../lib/format";
import { buildThreadRows, hasUnreadIncoming, prependPage, type ThreadRow } from "../lib/thread";
import { MESSAGE_MAX } from "../lib/validation";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Avatar, Notice, Text, colors, radii, spacing, typography } from "../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "Thread">;

export default function ThreadScreen({ navigation, route }: Props) {
  const { matchId, profile } = route.params;
  const { user } = useAuth();
  const { connected } = useRealtime();
  const queryClient = useQueryClient();
  const focused = useIsFocused();
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const [sendError, setSendError] = useState<string | null>(null);
  const [loadingOlder, setLoadingOlder] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  const meId = user?.id ?? "";
  const key = queryKeys.messages(matchId);

  const query = useQuery<MessagesPage>({
    queryKey: key,
    queryFn: () => listMessages(matchId),
    staleTime: 0,
    refetchInterval: connected ? false : 8_000
  });

  const safety = useSafetyActions({ userId: profile.id, name: profile.firstName, matchId, onGone: () => navigation.goBack() });

  const messages = query.data?.messages;
  const rows = useMemo(() => (messages ? buildThreadRows(messages, meId, formatDayLabel) : []), [messages, meId]);

  // Opening the thread (or receiving while it is open) marks incoming messages as read.
  const needsRead = messages ? hasUnreadIncoming(messages, meId) : false;
  useEffect(() => {
    if (!focused || !needsRead) {
      return;
    }
    markConversationRead(matchId)
      .then(() => {
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        void queryClient.invalidateQueries({ queryKey: queryKeys.unread });
        void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
        queryClient.setQueryData<MessagesPage>(key, (page) =>
          page
            ? {
                ...page,
                messages: page.messages.map((m) =>
                  m.senderId !== meId && m.readAt === null ? { ...m, readAt: new Date().toISOString() } : m
                )
              }
            : page
        );
      })
      .catch(() => undefined);
  }, [focused, key, matchId, meId, needsRead, queryClient]);

  const loadOlder = useCallback(async () => {
    const page = queryClient.getQueryData<MessagesPage>(key);
    if (!page?.hasMore || loadingOlder || page.messages.length === 0) {
      return;
    }
    setLoadingOlder(true);
    try {
      const older = await listMessages(matchId, page.messages[0].id);
      queryClient.setQueryData<MessagesPage>(key, (current) =>
        current ? { hasMore: older.hasMore, messages: prependPage(current.messages, older.messages) } : older
      );
    } catch {
      // The user can scroll again to retry.
    } finally {
      setLoadingOlder(false);
    }
  }, [key, loadingOlder, matchId, queryClient]);

  const send = async () => {
    const body = draft.trim();
    if (!body || sending) {
      return;
    }
    setSending(true);
    setSendError(null);
    setDraft("");
    try {
      const message = await sendMessage(matchId, body);
      queryClient.setQueryData<MessagesPage>(key, (page) =>
        page && !page.messages.some((m) => m.id === message.id) ? { ...page, messages: [...page.messages, message] } : page
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    } catch (error) {
      setDraft(body); // never lose what the user typed
      setSendError(friendlyError(error, "Message non envoyé. Réessayez."));
    } finally {
      setSending(false);
    }
  };

  const menu: SheetAction[] = [
    { label: "Voir le profil", onPress: () => navigation.navigate("ProfileDetail", { userId: profile.id, source: "match", matchId }) },
    { label: "Supprimer la conversation", onPress: safety.confirmClear },
    { label: "Signaler", onPress: safety.openReport },
    { label: "Bloquer", destructive: true, onPress: safety.confirmBlock },
    { label: "Annuler le match", destructive: true, onPress: safety.confirmUnmatch }
  ];

  const gone = query.error instanceof ApiError && query.error.status === 404;
  const photo = profile.photos[0];

  return (
    <SafeAreaLayout edges={["top", "left", "right", "bottom"]}>
      <View style={styles.header}>
        <Pressable accessibilityLabel="Retour" accessibilityRole="button" hitSlop={8} onPress={() => navigation.goBack()}>
          <Ionicons color={colors.text} name="chevron-back" size={26} />
        </Pressable>
        <Pressable
          accessibilityLabel={`Voir le profil de ${profile.firstName}`}
          accessibilityRole="button"
          onPress={() => navigation.navigate("ProfileDetail", { userId: profile.id, source: "match", matchId })}
          style={styles.who}
        >
          <Avatar name={profile.firstName} size={40} uri={photo ? absoluteUrl(photo.url) : null} />
          <Text variant="heading" weight="bold">
            {profile.firstName}
          </Text>
        </Pressable>
        <Pressable
          accessibilityLabel="Plus d'options"
          accessibilityRole="button"
          hitSlop={8}
          onPress={() => setMenuOpen(true)}
          testID="thread-menu"
        >
          <Ionicons color={colors.text} name="ellipsis-horizontal" size={24} />
        </Pressable>
      </View>

      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        {query.isLoading ? <LoadingView label="Chargement des messages…" /> : null}
        {gone ? (
          <ErrorView actionLabel="Retour" message="Cette conversation n'existe plus." onAction={() => navigation.goBack()} />
        ) : query.isError && !query.data ? (
          <ErrorView message="Impossible de charger les messages." onAction={() => void query.refetch()} />
        ) : null}

        {query.data ? (
          <FlatList<ThreadRow>
            contentContainerStyle={styles.messages}
            data={rows}
            inverted
            keyboardShouldPersistTaps="handled"
            keyExtractor={(row) => row.key}
            ListFooterComponent={
              loadingOlder ? (
                <Text style={styles.center} tone="muted" variant="caption">
                  Chargement des messages précédents…
                </Text>
              ) : rows.length === 0 ? (
                <Text style={styles.center} tone="muted">
                  C'est un match ! Envoyez le premier message à {profile.firstName}.
                </Text>
              ) : null
            }
            onEndReached={() => void loadOlder()}
            onEndReachedThreshold={0.4}
            renderItem={({ item }) => (item.kind === "day" ? <DaySeparator label={item.label} /> : <Bubble row={item} />)}
            testID="thread-list"
          />
        ) : null}

        {sendError ? <Notice style={styles.notice} title={sendError} tone="danger" /> : null}
        {!gone ? (
          <View style={styles.composer}>
            <TextInput
              accessibilityLabel="Votre message"
              maxLength={MESSAGE_MAX}
              multiline
              onChangeText={setDraft}
              placeholder="Écrire un message…"
              placeholderTextColor={colors.textSubtle}
              style={styles.input}
              testID="message-input"
              value={draft}
            />
            <Pressable
              accessibilityLabel="Envoyer"
              accessibilityRole="button"
              disabled={!draft.trim() || sending}
              onPress={() => void send()}
              style={[styles.send, !draft.trim() || sending ? styles.sendDisabled : null]}
              testID="send-message"
            >
              <Ionicons color={colors.inverse} name="arrow-up" size={20} />
            </Pressable>
          </View>
        ) : null}
      </KeyboardAvoidingView>

      <ActionSheet actions={menu} onClose={() => setMenuOpen(false)} title={profile.firstName} visible={menuOpen} />
      <ReportSheet
        name={profile.firstName}
        onClose={safety.closeReport}
        onSubmit={safety.submitReport}
        submitting={safety.reporting}
        visible={safety.reportOpen}
      />
    </SafeAreaLayout>
  );
}

function DaySeparator({ label }: { label: string }) {
  return (
    <Text style={styles.day} tone="muted" variant="caption" weight="semibold">
      {label}
    </Text>
  );
}

function Bubble({ row }: { row: Extract<ThreadRow, { kind: "message" }> }) {
  const { message, mine, showReceipt } = row;
  return (
    <View style={[styles.bubbleWrap, mine ? styles.mineWrap : styles.theirsWrap]}>
      <View accessibilityLabel={`${mine ? "Vous" : "Eux"} : ${message.body}`} style={[styles.bubble, mine ? styles.mine : styles.theirs]}>
        <Text tone={mine ? "inverse" : "default"}>{message.body}</Text>
      </View>
      <Text tone="muted" variant="caption">
        {formatClock(message.createdAt)}
        {showReceipt ? " · Lu" : ""}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
    gap: spacing.md
  },
  who: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.md },
  messages: { padding: spacing.lg, gap: spacing.sm },
  center: { textAlign: "center", padding: spacing.lg },
  day: { textAlign: "center", marginVertical: spacing.md, textTransform: "capitalize" },
  bubbleWrap: { maxWidth: "80%", gap: 2 },
  mineWrap: { alignSelf: "flex-end", alignItems: "flex-end" },
  theirsWrap: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { borderRadius: radii.lg, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  mine: { backgroundColor: colors.primary, borderBottomRightRadius: radii.xs },
  theirs: { backgroundColor: colors.backgroundElevated, borderWidth: 1, borderColor: colors.border, borderBottomLeftRadius: radii.xs },
  notice: { marginHorizontal: spacing.lg },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: 1,
    borderTopColor: colors.border,
    backgroundColor: colors.background
  },
  input: {
    flex: 1,
    maxHeight: 120,
    minHeight: 44,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radii.xl,
    backgroundColor: colors.backgroundElevated,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    color: colors.text,
    ...typography.body
  },
  send: {
    width: 44,
    height: 44,
    borderRadius: radii.pill,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center"
  },
  sendDisabled: { opacity: 0.4 }
});
