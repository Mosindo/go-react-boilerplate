import React, { useCallback, useEffect, useMemo, useState } from "react";
import { ActivityIndicator, FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useFocusEffect } from "@react-navigation/native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { errorMessage } from "../../api/client";
import { matchesApi } from "../../api/endpoints";
import type { ChatMessage } from "../../api/types";
import { useMatch, useMessages } from "../../hooks/queries";
import { keys, queryClient } from "../../hooks/queryClient";
import { useMyId } from "../../hooks/useAuth";
import { onMatchRemoved, setActiveChat, upsertMessage, useRealtimeConnected } from "../../hooks/useRealtime";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, spacing, typography } from "../../theme/palette";
import { Avatar, ErrorState, Icon, IconButton, ListRow, Loading, Screen, Sheet, Text, toast } from "../../ui";
import { clientId, daysBetween, formatDayLabel, formatTime } from "../../utils/format";
import type { AppScreenProps } from "../../navigation/types";
import { ReportSheet } from "../safety/ReportSheet";
import { useSafetyActions } from "../safety/useSafetyActions";

type Row = { kind: "message"; message: ChatMessage; showReadReceipt: boolean } | { kind: "day"; label: string; key: string };

export function ChatScreen({ navigation, route }: AppScreenProps<"Chat">) {
  const { matchId } = route.params;
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const myId = useMyId();
  const connected = useRealtimeConnected();
  const match = useMatch(matchId);
  const messages = useMessages(matchId, !connected);
  const [draft, setDraft] = useState("");
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);
  const other = match.data?.user;
  const safety = useSafetyActions(() => {
    setReporting(false);
    navigation.goBack();
  });

  // Mark as read while the conversation is on screen.
  useFocusEffect(
    useCallback(() => {
      setActiveChat(matchId);
      matchesApi
        .markRead(matchId)
        .then(() => queryClient.invalidateQueries({ queryKey: keys.matches }))
        .catch(() => undefined);
      return () => setActiveChat(null);
    }, [matchId])
  );

  useEffect(
    () =>
      onMatchRemoved((removed) => {
        if (removed === matchId) {
          toast("Cette conversation n'est plus disponible.", "info");
          navigation.goBack();
        }
      }),
    [matchId, navigation]
  );

  const rows = useMemo<Row[]>(() => {
    const list = messages.data?.pages.flatMap((p) => p.messages) ?? [];
    // `list` is newest first (inverted list); insert day separators after the
    // last message of each day, and a "seen" receipt on my newest read message.
    const out: Row[] = [];
    const newestReadMine = list.find((m) => m.senderUserId === myId && m.isRead);
    list.forEach((m, i) => {
      out.push({ kind: "message", message: m, showReadReceipt: newestReadMine?.id === m.id && list[0]?.senderUserId === myId });
      const older = list[i + 1];
      if (!older || daysBetween(new Date(older.createdAt), new Date(m.createdAt)) !== 0) {
        out.push({ kind: "day", label: formatDayLabel(m.createdAt), key: `day-${m.id}` });
      }
    });
    return out;
  }, [messages.data, myId]);

  const send = async (content: string, existingClientId?: string) => {
    const trimmed = content.trim();
    if (!trimmed) {
      return;
    }
    const cid = existingClientId ?? clientId();
    const optimistic: ChatMessage = {
      id: `local-${cid}`,
      matchId,
      senderUserId: myId,
      content: trimmed,
      clientId: cid,
      createdAt: new Date().toISOString(),
      isRead: false,
      pending: true
    };
    upsertMessage(matchId, optimistic);
    try {
      const saved = await matchesApi.send(matchId, trimmed, cid);
      upsertMessage(matchId, saved);
      void queryClient.invalidateQueries({ queryKey: keys.matches });
    } catch (e) {
      upsertMessage(matchId, { ...optimistic, pending: false, failed: true });
      toast(errorMessage(e), "error");
    }
  };

  const submit = () => {
    const content = draft;
    setDraft("");
    void send(content);
  };

  if (match.isLoading && !match.data) {
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  }
  if (match.error || !other) {
    return (
      <Screen>
        <ErrorState message={match.error ? errorMessage(match.error) : "Conversation introuvable."} onRetry={() => navigation.goBack()} />
      </Screen>
    );
  }

  return (
    <Screen padded={false} edges={["top"]} testID="chat-screen">
      <View style={[styles.header, { borderBottomColor: colors.border }]}>
        <IconButton icon="chevron-back" accessibilityLabel="Retour" onPress={() => navigation.goBack()} />
        <Pressable
          style={styles.headerId}
          onPress={() => navigation.navigate("ProfileDetail", { userId: other.userId, fromChat: true })}
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${other.firstName}`}
        >
          <Avatar url={other.photos[0]?.url} name={other.firstName} size={38} />
          <View>
            <Text variant="bodyStrong">{other.firstName}</Text>
            {!connected ? (
              <Text variant="caption" tone="subtle">
                Reconnexion…
              </Text>
            ) : null}
          </View>
        </Pressable>
        <IconButton icon="ellipsis-horizontal" accessibilityLabel="Options de la conversation" onPress={() => setMenu(true)} testID="chat-menu" />
      </View>

      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={insets.top + 52}>
        {messages.isLoading ? (
          <Loading />
        ) : messages.error && rows.length === 0 ? (
          <ErrorState message={errorMessage(messages.error)} onRetry={() => void messages.refetch()} />
        ) : (
          <FlatList
            inverted
            data={rows}
            keyExtractor={(r) => (r.kind === "day" ? r.key : r.message.clientId ? `c-${r.message.clientId}` : r.message.id)}
            contentContainerStyle={styles.list}
            onEndReachedThreshold={0.3}
            onEndReached={() => {
              if (messages.hasNextPage && !messages.isFetchingNextPage) void messages.fetchNextPage();
            }}
            ListFooterComponent={
              messages.isFetchingNextPage ? (
                <ActivityIndicator color={colors.primary} />
              ) : !messages.hasNextPage ? (
                <View style={styles.intro}>
                  <Avatar url={other.photos[0]?.url} name={other.firstName} size={72} />
                  <Text variant="heading">Vous avez matché !</Text>
                  <Text tone="muted" align="center">
                    Restez bienveillant·es. Tu peux signaler ou bloquer à tout moment depuis le menu « … ».
                  </Text>
                </View>
              ) : null
            }
            renderItem={({ item }) => {
              if (item.kind === "day") {
                return (
                  <Text variant="caption" tone="subtle" align="center" style={styles.day}>
                    {item.label}
                  </Text>
                );
              }
              const m = item.message;
              const mine = m.senderUserId === myId;
              return (
                <View style={[styles.bubbleRow, mine ? styles.right : styles.left]}>
                  <Pressable
                    disabled={!m.failed}
                    onPress={() => void send(m.content, m.clientId)}
                    accessibilityLabel={m.failed ? "Échec de l'envoi, toucher pour réessayer" : undefined}
                    style={[
                      styles.bubble,
                      mine
                        ? { backgroundColor: colors.primary, borderBottomRightRadius: 6 }
                        : { backgroundColor: colors.surface, borderColor: colors.border, borderWidth: StyleSheet.hairlineWidth, borderBottomLeftRadius: 6 },
                      m.pending ? styles.pending : null
                    ]}
                  >
                    <Text style={{ color: mine ? colors.onPrimary : colors.text }} selectable>
                      {m.content}
                    </Text>
                    <Text variant="caption" style={[styles.time, { color: mine ? colors.onPrimary : colors.textSubtle }]}>
                      {formatTime(m.createdAt)}
                    </Text>
                  </Pressable>
                  {m.failed ? (
                    <View style={styles.status}>
                      <Icon name="alert-circle" size={14} color={colors.danger} />
                      <Text variant="caption" tone="danger">
                        Non envoyé · toucher pour réessayer
                      </Text>
                    </View>
                  ) : item.showReadReceipt ? (
                    <Text variant="caption" tone="subtle" style={styles.status} testID="read-receipt">
                      Vu
                    </Text>
                  ) : null}
                </View>
              );
            }}
          />
        )}

        <View style={[styles.composer, { borderTopColor: colors.border, backgroundColor: colors.background, paddingBottom: insets.bottom + spacing.xs }]}>
          <TextInput
            value={draft}
            onChangeText={setDraft}
            placeholder={`Écrire à ${other.firstName}…`}
            placeholderTextColor={colors.textSubtle}
            multiline
            maxLength={2000}
            accessibilityLabel="Message"
            style={[typography.body, styles.input, { color: colors.text, backgroundColor: colors.surface, borderColor: colors.border }]}
            testID="chat-input"
          />
          <IconButton
            icon="arrow-up"
            variant="primary"
            size={44}
            accessibilityLabel="Envoyer"
            disabled={!draft.trim()}
            onPress={submit}
            testID="chat-send"
          />
        </View>
      </KeyboardAvoidingView>

      <Sheet visible={menu} title={other.firstName} onClose={() => setMenu(false)}>
        <ListRow icon="person-outline" title="Voir le profil" onPress={() => { setMenu(false); navigation.navigate("ProfileDetail", { userId: other.userId, fromChat: true }); }} />
        <ListRow icon="heart-dislike-outline" title="Annuler le match" onPress={() => { setMenu(false); void safety.confirmUnmatch(matchId, other.firstName); }} testID="chat-unmatch" />
        <ListRow icon="flag-outline" title="Signaler" destructive onPress={() => { setMenu(false); setReporting(true); }} testID="chat-report" />
        <ListRow icon="ban-outline" title="Bloquer" destructive onPress={() => { setMenu(false); void safety.confirmBlock(other.userId, other.firstName); }} testID="chat-block" />
      </Sheet>
      <ReportSheet
        visible={reporting}
        name={other.firstName}
        loading={safety.report.isPending}
        onClose={() => setReporting(false)}
        onSubmit={(input) => safety.report.mutate({ userId: other.userId, ...input })}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { flexDirection: "row", alignItems: "center", paddingHorizontal: spacing.xs, paddingBottom: spacing.xs, borderBottomWidth: StyleSheet.hairlineWidth },
  headerId: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.sm },
  list: { paddingHorizontal: spacing.md, paddingVertical: spacing.md, gap: 4, maxWidth: 760, width: "100%", alignSelf: "center" },
  intro: { alignItems: "center", gap: spacing.xs, paddingVertical: spacing.xl, paddingHorizontal: spacing.lg },
  day: { marginVertical: spacing.sm },
  bubbleRow: { maxWidth: "82%", marginVertical: 2 },
  left: { alignSelf: "flex-start" },
  right: { alignSelf: "flex-end", alignItems: "flex-end" },
  bubble: { paddingHorizontal: 14, paddingVertical: 9, borderRadius: radii.lg },
  pending: { opacity: 0.6 },
  time: { fontSize: 11, opacity: 0.75, marginTop: 2, alignSelf: "flex-end" },
  status: { flexDirection: "row", alignItems: "center", gap: 4, marginTop: 2 },
  composer: { flexDirection: "row", alignItems: "flex-end", gap: spacing.xs, paddingHorizontal: spacing.md, paddingTop: spacing.xs, borderTopWidth: StyleSheet.hairlineWidth },
  input: { flex: 1, maxHeight: 140, minHeight: 44, borderWidth: 1, borderRadius: 22, paddingHorizontal: spacing.md, paddingTop: 10, paddingBottom: 10 }
});
