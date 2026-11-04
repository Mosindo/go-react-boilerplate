import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { errorMessage, isApiError } from "../../api/client";
import { chatApi } from "../../api/endpoints";
import type { Message } from "../../api/types";
import { addMessageToCache } from "../../hooks/realtime";
import { keys, useMessages } from "../../hooks/queries";
import { useSession } from "../../hooks/session";
import type { MainStackParams } from "../../navigation/types";
import { confirm } from "../../shared/feedback/confirm";
import { Button, ErrorState, Loader, Sheet, Text, radius, spacing, useColors, useToast } from "../../shared/ui";
import { ReportSheet } from "../discover/ReportSheet";
import { useBlock } from "../discover/useSafetyActions";
import { flattenPages, isReadByPeer, latestOwnMessageId } from "../../utils/messages";
import { formatHour } from "../../utils/time";
import { MAX_MESSAGE_LENGTH, cleanMessage } from "../../utils/validation";

type Pending = { tmpId: string; body: string; failed: boolean; createdAt: string };

export function ChatScreen({ route, navigation }: NativeStackScreenProps<MainStackParams, "Chat">) {
  const { conversationId, name, userId } = route.params;
  const c = useColors();
  const qc = useQueryClient();
  const toast = useToast();
  const insets = useSafeAreaInsets();
  const { user } = useSession();
  const myId = user?.id ?? "";
  const query = useMessages(conversationId);
  const [draft, setDraft] = useState("");
  const [pending, setPending] = useState<Pending[]>([]);
  const [menu, setMenu] = useState(false);
  const [report, setReport] = useState(false);
  const counter = useRef(0);
  const askBlock = useBlock(userId, name, () => navigation.popToTop());

  const messages = useMemo(() => (query.data ? flattenPages(query.data.pages) : []), [query.data]);
  const peerLastRead = query.data?.pages[0]?.peerLastReadId ?? "0";
  const lastOwn = latestOwnMessageId(messages, myId);
  const newestId = messages[0]?.id;
  const newestFromPeer = messages[0] !== undefined && messages[0].senderId !== myId;

  useLayoutEffect(() => {
    navigation.setOptions({
      title: name,
      headerRight: () => <Button title="Options" variant="ghost" onPress={() => setMenu(true)} testID="chat-menu" />,
    });
  }, [navigation, name]);

  const markRead = useCallback(() => {
    chatApi.markRead(conversationId).then(() => qc.invalidateQueries({ queryKey: keys.conversations })).catch(() => undefined);
  }, [conversationId, qc]);

  // Mark as read when opened and whenever a message from the other person is the newest.
  useEffect(() => {
    if (query.isSuccess && (newestFromPeer || newestId === undefined)) markRead();
  }, [query.isSuccess, newestFromPeer, newestId, markRead]);

  const sendBody = useCallback(
    async (tmpId: string, body: string) => {
      try {
        const saved: Message = await chatApi.send(conversationId, body);
        addMessageToCache(qc, saved);
        setPending((p) => p.filter((x) => x.tmpId !== tmpId));
        void qc.invalidateQueries({ queryKey: keys.conversations });
      } catch (err) {
        setPending((p) => p.map((x) => (x.tmpId === tmpId ? { ...x, failed: true } : x)));
        toast.show(errorMessage(err), "error");
      }
    },
    [conversationId, qc, toast],
  );

  const send = () => {
    const cleaned = cleanMessage(draft);
    if (!cleaned.ok) return;
    counter.current += 1;
    const tmpId = `tmp-${counter.current}`;
    setPending((p) => [{ tmpId, body: cleaned.body, failed: false, createdAt: new Date().toISOString() }, ...p]);
    setDraft("");
    void sendBody(tmpId, cleaned.body);
  };

  const retry = (item: Pending) => {
    setPending((p) => p.map((x) => (x.tmpId === item.tmpId ? { ...x, failed: false } : x)));
    void sendBody(item.tmpId, item.body);
  };

  if (query.isPending) return <Loader />;
  if (query.isError) {
    const gone = isApiError(query.error) && query.error.status === 404;
    return <ErrorState message={gone ? "Cette conversation n'existe plus." : errorMessage(query.error)} onRetry={gone ? () => navigation.goBack() : () => void query.refetch()} />;
  }

  const rows: (Message | Pending)[] = [...pending, ...messages];

  return (
    <KeyboardAvoidingView style={{ flex: 1, backgroundColor: c.background }} behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={Platform.OS === "ios" ? 90 : 0}>
      <FlatList
        inverted
        data={rows}
        keyExtractor={(m) => ("tmpId" in m ? m.tmpId : m.id)}
        contentContainerStyle={{ padding: spacing.lg, gap: spacing.sm }}
        onEndReached={() => { if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage(); }}
        onEndReachedThreshold={0.5}
        ListEmptyComponent={<View style={styles.empty}><Text tone="muted" style={{ textAlign: "center" }}>C'est un match ! Écrivez le premier message à {name}.</Text></View>}
        ListFooterComponent={query.isFetchingNextPage ? <Loader /> : null}
        renderItem={({ item }) => {
          const isPending = "tmpId" in item;
          const mine = isPending || item.senderId === myId;
          const failed = isPending && item.failed;
          return (
            <View style={[styles.bubbleWrap, mine ? styles.right : styles.left]}>
              <Pressable
                disabled={!failed}
                onPress={() => isPending && retry(item)}
                accessibilityLabel={failed ? `Message non envoyé : ${item.body}. Toucher pour réessayer` : item.body}
                style={[styles.bubble, mine ? { backgroundColor: c.primary } : { backgroundColor: c.surface, borderColor: c.border, borderWidth: 1 }, failed ? { opacity: 0.6 } : null]}
              >
                <Text style={{ color: mine ? c.onPrimary : c.text }}>{item.body}</Text>
              </Pressable>
              <Text variant="caption" tone={failed ? "danger" : "muted"}>
                {failed ? "Non envoyé — toucher pour réessayer" : isPending ? "Envoi…" : formatHour(item.createdAt)}
                {!isPending && item.id === lastOwn && isReadByPeer(item.id, peerLastRead) ? " · Lu" : ""}
              </Text>
            </View>
          );
        }}
      />
      <View style={[styles.composer, { borderTopColor: c.border, backgroundColor: c.surface, paddingBottom: Math.max(insets.bottom, spacing.sm) }]}>
        <TextInput
          testID="chat-input"
          accessibilityLabel="Votre message"
          value={draft}
          onChangeText={setDraft}
          placeholder="Écrire un message…"
          placeholderTextColor={c.textMuted}
          multiline
          maxLength={MAX_MESSAGE_LENGTH}
          style={[styles.input, { color: c.text, backgroundColor: c.background, borderColor: c.border }]}
        />
        <Button title="Envoyer" onPress={send} disabled={!draft.trim()} testID="chat-send" />
      </View>

      <Sheet visible={menu} title={name} onClose={() => setMenu(false)}>
        <Button title="Voir le profil" variant="secondary" onPress={() => { setMenu(false); navigation.navigate("ProfileDetail", { userId, conversationId }); }} />
        <Button title="Annuler le match" variant="secondary" onPress={() => { setMenu(false); confirm("Annuler le match ?", `La conversation avec ${name} sera supprimée pour vous deux.`, "Annuler le match", () => {
          chatApi.unmatch(conversationId).then(() => { void qc.invalidateQueries({ queryKey: keys.conversations }); navigation.popToTop(); }).catch((err: unknown) => toast.show(errorMessage(err), "error"));
        }); }} />
        <Button title="Signaler" variant="secondary" onPress={() => { setMenu(false); setReport(true); }} />
        <Button title="Bloquer" variant="danger" onPress={() => { setMenu(false); askBlock(); }} testID="chat-block" />
      </Sheet>
      <ReportSheet userId={userId} visible={report} onClose={() => setReport(false)} onReported={() => { setReport(false); navigation.popToTop(); }} />
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  empty: { padding: spacing.xl, transform: [{ scaleY: -1 }] },
  bubbleWrap: { maxWidth: "82%", gap: 2 },
  right: { alignSelf: "flex-end", alignItems: "flex-end" },
  left: { alignSelf: "flex-start", alignItems: "flex-start" },
  bubble: { borderRadius: radius.lg, paddingHorizontal: spacing.md, paddingVertical: spacing.sm },
  composer: { flexDirection: "row", alignItems: "flex-end", gap: spacing.sm, padding: spacing.sm, borderTopWidth: StyleSheet.hairlineWidth },
  input: { flex: 1, minHeight: 46, maxHeight: 120, borderRadius: radius.lg, borderWidth: 1, paddingHorizontal: spacing.md, paddingVertical: spacing.sm, fontSize: 16 },
});
