import React, { useCallback, useEffect, useMemo, useState } from "react";
import { FlatList, KeyboardAvoidingView, Modal, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useFocusEffect } from "@react-navigation/native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { isApiError } from "../../api/client";
import { chatApi, matchesApi, safetyApi } from "../../api/dating";
import type { Message } from "../../api/types";
import { setActiveConversation } from "../../hooks/presence";
import { keys, useMessages, useSendMessage } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import type { AppStackParamList } from "../../navigation/types";
import { ConfirmDialog, EmptyView, ErrorView, LoadingView, showToast } from "../../shared/feedback";
import { Button, IconButton, Loader, Text, colors, radii, spacing, typography } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";
import { formatRelative, formatTime } from "../../utils/format";

type Props = NativeStackScreenProps<AppStackParamList, "Chat">;

export default function ChatScreen({ navigation, route }: Props) {
  const { conversationId, firstName, matchId, userId } = route.params;
  const { user } = useAuth();
  const client = useQueryClient();
  const query = useMessages(conversationId);
  const send = useSendMessage(conversationId);
  const [draft, setDraft] = useState("");
  const [menu, setMenu] = useState(false);
  const [confirm, setConfirm] = useState<"unmatch" | "block" | null>(null);

  const messages = useMemo(() => query.data?.pages.flatMap((p) => p.messages) ?? [], [query.data]);
  const otherLastReadAt = query.data?.pages[0]?.otherLastReadAt ?? null;
  const latestIncoming = messages.find((m) => m.senderUserId !== user?.id)?.id;
  const lastMineId = messages.find((m) => m.senderUserId === user?.id)?.id;

  const markRead = useCallback(() => {
    chatApi
      .markRead(conversationId)
      .then(() => {
        void client.invalidateQueries({ queryKey: keys.matches });
        void client.invalidateQueries({ queryKey: keys.unread });
      })
      .catch(() => undefined);
  }, [client, conversationId]);

  // Read receipts: on open and whenever a new incoming message arrives.
  useFocusEffect(
    useCallback(() => {
      setActiveConversation(conversationId);
      markRead();
      return () => setActiveConversation(null);
    }, [conversationId, markRead])
  );
  useEffect(() => {
    if (latestIncoming) {
      markRead();
    }
  }, [latestIncoming, markRead]);

  const leave = useMutation({
    mutationFn: async (kind: "unmatch" | "block") => (kind === "unmatch" ? matchesApi.unmatch(matchId) : safetyApi.block(userId)),
    onSuccess: (_, kind) => {
      void client.invalidateQueries({ queryKey: keys.matches });
      client.removeQueries({ queryKey: keys.messages(conversationId) });
      showToast(kind === "unmatch" ? `Match avec ${firstName} supprimé` : `${firstName} est bloqué·e`);
      navigation.goBack();
    },
    onError: (e) => showToast(errorMessage(e), "danger")
  });

  const submit = () => {
    const content = draft.trim();
    if (!content || send.isPending) {
      return;
    }
    send.mutate(content, {
      onSuccess: () => setDraft(""),
      onError: (e) => showToast(errorMessage(e), "danger")
    });
  };

  const gone = query.isError && isApiError(query.error, "not_found");

  const renderItem = ({ item, index }: { item: Message; index: number }) => {
    const mine = item.senderUserId === user?.id;
    const older = messages[index + 1];
    const showTime = !older || new Date(item.createdAt).getTime() - new Date(older.createdAt).getTime() > 15 * 60_000;
    const read = mine && otherLastReadAt !== null && new Date(otherLastReadAt) >= new Date(item.createdAt);
    return (
      <View>
        {showTime ? (
          <Text align="center" style={styles.timeDivider} tone="subtle" variant="caption">
            {formatRelative(item.createdAt)}
          </Text>
        ) : null}
        <View style={[styles.bubbleRow, mine ? styles.mineRow : null]}>
          <View style={[styles.bubble, mine ? styles.mine : styles.theirs]}>
            <Text style={mine ? styles.mineText : undefined}>{item.content}</Text>
            <Text style={[styles.meta, mine ? styles.mineMeta : null]} tone="subtle" variant="caption">
              {formatTime(item.createdAt)}
            </Text>
          </View>
        </View>
        {item.id === lastMineId ? (
          <Text style={styles.receipt} tone="subtle" testID="read-receipt" variant="caption">
            {read ? "Lu" : "Envoyé"}
          </Text>
        ) : null}
      </View>
    );
  };

  return (
    <SafeAreaView edges={["top", "left", "right", "bottom"]} style={styles.safe} testID="chat-screen">
      <View style={styles.header}>
        <IconButton accessibilityLabel="Retour" glyph="‹" onPress={() => navigation.goBack()} size={40} />
        <Pressable accessibilityRole="button" onPress={() => navigation.navigate("ProfileDetail", { userId })} style={styles.headerTitle}>
          <Text numberOfLines={1} variant="heading">
            {firstName}
          </Text>
          <Text tone="muted" variant="caption">
            Voir le profil
          </Text>
        </Pressable>
        <IconButton accessibilityLabel="Options de la conversation" glyph="⋯" onPress={() => setMenu(true)} size={40} testID="chat-menu" />
      </View>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        {query.isLoading ? (
          <LoadingView />
        ) : gone ? (
          <EmptyView message="Cette conversation n'est plus disponible." title="Conversation terminée" />
        ) : query.isError ? (
          <ErrorView onRetry={() => void query.refetch()} />
        ) : (
          <FlatList
            ListEmptyComponent={
              <View style={styles.emptyChat}>
                <Text align="center" variant="heading">
                  Vous avez matché !
                </Text>
                <Text align="center" tone="muted">
                  Un petit mot sur un centre d&apos;intérêt commun est souvent un bon début.
                </Text>
              </View>
            }
            ListFooterComponent={query.isFetchingNextPage ? <Loader /> : null}
            contentContainerStyle={styles.list}
            data={messages}
            inverted={messages.length > 0}
            keyExtractor={(m) => m.id}
            onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
            onEndReachedThreshold={0.3}
            renderItem={renderItem}
          />
        )}
        {!gone ? (
          <View style={styles.composer}>
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
              placeholder={`Message à ${firstName}…`}
              placeholderTextColor={colors.textSubtle}
              selectionColor={colors.primary}
              style={styles.input}
              testID="chat-input"
              value={draft}
            />
            <IconButton
              accessibilityLabel="Envoyer"
              disabled={!draft.trim() || send.isPending}
              filled
              glyph="↑"
              onPress={submit}
              size={46}
              testID="chat-send"
              tone="primary"
            />
          </View>
        ) : null}
      </KeyboardAvoidingView>

      <Modal animationType="fade" onRequestClose={() => setMenu(false)} transparent visible={menu}>
        <Pressable onPress={() => setMenu(false)} style={styles.backdrop}>
          <Pressable style={styles.sheet}>
            <Button
              label="Voir le profil"
              onPress={() => {
                setMenu(false);
                navigation.navigate("ProfileDetail", { userId });
              }}
              variant="secondary"
            />
            <Button
              label="Supprimer le match"
              onPress={() => {
                setMenu(false);
                setConfirm("unmatch");
              }}
              testID="chat-unmatch"
              variant="secondary"
            />
            <Button
              label="Bloquer"
              onPress={() => {
                setMenu(false);
                setConfirm("block");
              }}
              testID="chat-block"
              variant="danger"
            />
            <Button
              label="Signaler"
              onPress={() => {
                setMenu(false);
                navigation.navigate("Report", { userId, firstName });
              }}
              testID="chat-report"
              variant="danger"
            />
            <Button label="Fermer" onPress={() => setMenu(false)} variant="ghost" />
          </Pressable>
        </Pressable>
      </Modal>
      <ConfirmDialog
        confirmLabel={confirm === "block" ? "Bloquer" : "Supprimer"}
        destructive
        loading={leave.isPending}
        message={
          confirm === "block"
            ? `${firstName} ne pourra plus te voir ni t'écrire. La conversation sera supprimée. ${firstName} ne sera pas prévenu·e.`
            : `La conversation avec ${firstName} sera supprimée pour vous deux, et vous ne vous verrez plus dans Découvrir.`
        }
        onCancel={() => setConfirm(null)}
        onConfirm={() => confirm && leave.mutate(confirm)}
        title={confirm === "block" ? `Bloquer ${firstName} ?` : "Supprimer ce match ?"}
        visible={confirm !== null}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: {
    flex: 1,
    backgroundColor: colors.background
  },
  flex: {
    flex: 1
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  headerTitle: {
    flex: 1
  },
  list: {
    flexGrow: 1,
    padding: spacing.lg,
    gap: spacing.xs
  },
  emptyChat: {
    flex: 1,
    justifyContent: "center",
    gap: spacing.sm,
    padding: spacing.xl
  },
  timeDivider: {
    marginVertical: spacing.md
  },
  bubbleRow: {
    flexDirection: "row"
  },
  mineRow: {
    justifyContent: "flex-end"
  },
  bubble: {
    maxWidth: "80%",
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radii.lg,
    gap: 2
  },
  mine: {
    backgroundColor: colors.primary,
    borderBottomRightRadius: 6
  },
  theirs: {
    backgroundColor: colors.surfaceRaised,
    borderBottomLeftRadius: 6
  },
  mineText: {
    color: colors.onPrimary
  },
  meta: {
    alignSelf: "flex-end",
    fontSize: 11
  },
  mineMeta: {
    color: "rgba(26, 14, 9, 0.65)"
  },
  receipt: {
    alignSelf: "flex-end",
    marginTop: 2,
    marginRight: spacing.xs
  },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: colors.border
  },
  input: {
    flex: 1,
    maxHeight: 140,
    minHeight: 46,
    borderRadius: radii.lg,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surfaceMuted,
    color: colors.text,
    paddingHorizontal: spacing.lg,
    paddingTop: 12,
    paddingBottom: 12,
    ...typography.body
  },
  backdrop: {
    flex: 1,
    backgroundColor: colors.overlay,
    justifyContent: "flex-end"
  },
  sheet: {
    width: "100%",
    maxWidth: 640,
    alignSelf: "center",
    backgroundColor: colors.backgroundElevated,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.xl,
    paddingBottom: spacing.xxxl,
    gap: spacing.sm
  }
});
