import React, { useCallback, useEffect, useMemo, useState } from "react";
import { FlatList, KeyboardAvoidingView, Platform, Pressable, StyleSheet, TextInput, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useFocusEffect } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage, isApiError } from "../../api/client";
import { chatApi } from "../../api/endpoints";
import type { Conversation, Message } from "../../api/types";
import type { MainStackParamList } from "../../navigation/types";
import { setActiveConversation } from "../../realtime/RealtimeProvider";
import { useAuth } from "../../state/AuthProvider";
import { keys } from "../../state/queryClient";
import { useMessages, useSendMessage } from "../../state/queries";
import { radii, spacing, type Palette } from "../../theme/palette";
import { useStyles, useTheme } from "../../theme/ThemeProvider";
import { Avatar, EmptyState, ErrorState, IconButton, LoadingState, Text, useToast } from "../../ui";
import { dayLabel, messageTime, sameDay } from "../../utils/format";
import { SafetyMenu } from "../safety/SafetyMenu";

type Props = NativeStackScreenProps<MainStackParamList, "Conversation">;

export function ConversationScreen({ navigation, route }: Props) {
  const { matchId, user: other } = route.params;
  const styles = useStyles(makeStyles);
  const { colors } = useTheme();
  const { user } = useAuth();
  const toast = useToast();
  const qc = useQueryClient();
  const query = useMessages(matchId);
  const send = useSendMessage(matchId);
  const [draft, setDraft] = useState("");
  const [menu, setMenu] = useState(false);

  const messages = useMemo(() => query.data?.pages.flatMap((p) => p.messages) ?? [], [query.data]);
  const otherReadAt = query.data?.pages[0]?.otherLastReadAt ?? null;
  const newestIncoming = messages.find((m) => m.senderId !== user?.id)?.id;

  const markRead = useCallback(() => {
    chatApi
      .markRead(matchId)
      .then(() => {
        qc.setQueryData<Conversation[]>(keys.conversations, (l) => l?.map((c) => (c.matchId === matchId ? { ...c, unreadCount: 0 } : c)));
        void qc.invalidateQueries({ queryKey: keys.notifications });
      })
      .catch(() => undefined);
  }, [matchId, qc]);

  useFocusEffect(
    useCallback(() => {
      setActiveConversation(matchId);
      return () => setActiveConversation(null);
    }, [matchId])
  );

  // Mark as read when opening and whenever a new incoming message arrives.
  useEffect(() => {
    if (query.isSuccess) markRead();
  }, [query.isSuccess, newestIncoming, markRead]);

  // The match disappeared (unmatch/block by the other member).
  useEffect(() => {
    if (query.isError && isApiError(query.error, "conversation_not_found")) {
      toast({ title: "Cette conversation n'est plus disponible", tone: "error" });
      navigation.goBack();
    }
  }, [navigation, query.error, query.isError, toast]);

  const submit = () => {
    const body = draft.trim();
    if (!body || send.isPending) return;
    setDraft("");
    send.mutate(body, {
      onError: (e) => {
        setDraft(body);
        toast({ title: "Message non envoyé", body: errorMessage(e), tone: "error" });
      }
    });
  };

  const openProfile = () => navigation.navigate("ProfileDetail", { userId: other.userId, profile: other, matchId });
  const lastMineId = messages.find((m) => m.senderId === user?.id)?.id;

  const renderItem = ({ item, index }: { item: Message; index: number }) => {
    const mine = item.senderId === user?.id;
    const older = messages[index + 1];
    const showDay = !older || !sameDay(older.createdAt, item.createdAt);
    const read = mine && otherReadAt !== null && new Date(otherReadAt) >= new Date(item.createdAt);
    return (
      <View>
        {showDay ? (
          <Text center style={styles.day} tone="subtle" variant="caption">
            {dayLabel(item.createdAt)}
          </Text>
        ) : null}
        <View style={[styles.bubbleRow, mine ? styles.right : styles.left]}>
          <View style={[styles.bubble, mine ? styles.mine : styles.theirs]}>
            <Text selectable style={{ color: mine ? colors.onPrimary : colors.text }}>
              {item.body}
            </Text>
          </View>
          <Text style={styles.meta} tone="subtle" variant="caption">
            {messageTime(item.createdAt)}
            {item.id === lastMineId ? (read ? " · Lu" : " · Envoyé") : ""}
          </Text>
        </View>
      </View>
    );
  };

  let content: React.ReactNode;
  if (query.isLoading) {
    content = <LoadingState />;
  } else if (query.isError) {
    content = <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />;
  } else if (!messages.length) {
    content = (
      <EmptyState
        body={`Un compliment sincère ou une question sur ${other.interests[0]?.label.toLowerCase() ?? "ses passions"} : lancez-vous !`}
        icon="chatbubble-ellipses-outline"
        title={`Dites bonjour à ${other.firstName}`}
      />
    );
  } else {
    content = (
      <FlatList
        contentContainerStyle={styles.list}
        data={messages}
        inverted
        keyExtractor={(m) => m.id}
        keyboardShouldPersistTaps="handled"
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && query.fetchNextPage()}
        onEndReachedThreshold={0.4}
        renderItem={renderItem}
        testID="message-list"
      />
    );
  }

  return (
    <SafeAreaView edges={["top", "bottom"]} style={[styles.flex, { backgroundColor: colors.background }]}>
      <View style={styles.header}>
        <IconButton icon="chevron-back" label="Retour" onPress={navigation.goBack} />
        <Pressable accessibilityLabel={`Voir le profil de ${other.firstName}`} accessibilityRole="button" onPress={openProfile} style={styles.who}>
          <Avatar photo={other.photos[0]} size={38} />
          <Text numberOfLines={1} variant="heading">
            {other.firstName}
          </Text>
        </Pressable>
        <IconButton icon="ellipsis-horizontal" label="Options de la conversation" onPress={() => setMenu(true)} testID="conversation-menu" />
      </View>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} keyboardVerticalOffset={8} style={styles.flex}>
        <View style={styles.flex}>{content}</View>
        <View style={styles.composer}>
          <TextInput
            accessibilityLabel="Votre message"
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
            placeholder="Écrire un message…"
            placeholderTextColor={colors.textSubtle}
            style={styles.input}
            testID="composer-input"
            value={draft}
          />
          <IconButton disabled={!draft.trim()} icon="arrow-up" label="Envoyer" onPress={submit} size={44} testID="composer-send" tone="primary" />
        </View>
      </KeyboardAvoidingView>
      <SafetyMenu
        firstName={other.firstName}
        matchId={matchId}
        onClose={() => setMenu(false)}
        onRemoved={() => navigation.goBack()}
        onViewProfile={() => {
          setMenu(false);
          openProfile();
        }}
        userId={other.userId}
        visible={menu}
      />
    </SafeAreaView>
  );
}

const makeStyles = (c: Palette) =>
  StyleSheet.create({
    flex: { flex: 1 },
    header: {
      flexDirection: "row",
      alignItems: "center",
      gap: spacing.xs,
      paddingHorizontal: spacing.xs,
      paddingBottom: spacing.xs,
      borderBottomWidth: StyleSheet.hairlineWidth,
      borderBottomColor: c.border
    },
    who: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.sm },
    list: { paddingHorizontal: spacing.md, paddingVertical: spacing.md, width: "100%", maxWidth: 720, alignSelf: "center" },
    day: { marginVertical: spacing.sm },
    bubbleRow: { marginVertical: 3, maxWidth: "80%" },
    left: { alignSelf: "flex-start", alignItems: "flex-start" },
    right: { alignSelf: "flex-end", alignItems: "flex-end" },
    bubble: { paddingHorizontal: spacing.md, paddingVertical: 10, borderRadius: radii.lg },
    mine: { backgroundColor: c.primary, borderBottomRightRadius: 6 },
    theirs: { backgroundColor: c.surface, borderBottomLeftRadius: 6, borderWidth: StyleSheet.hairlineWidth, borderColor: c.border },
    meta: { marginTop: 2, marginHorizontal: 4 },
    composer: {
      flexDirection: "row",
      alignItems: "flex-end",
      gap: spacing.xs,
      padding: spacing.sm,
      borderTopWidth: StyleSheet.hairlineWidth,
      borderTopColor: c.border,
      width: "100%",
      maxWidth: 720,
      alignSelf: "center"
    },
    input: {
      flex: 1,
      minHeight: 44,
      maxHeight: 130,
      borderRadius: 22,
      paddingHorizontal: spacing.md,
      paddingTop: 11,
      paddingBottom: 11,
      backgroundColor: c.surface,
      borderWidth: 1,
      borderColor: c.border,
      color: c.text,
      fontSize: 16
    }
  });
