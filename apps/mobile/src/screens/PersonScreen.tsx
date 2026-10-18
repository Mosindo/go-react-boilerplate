import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, errorMessage } from "../api/client";
import { unmatch, swipe } from "../api/discovery";
import { getProfile } from "../api/profile";
import { blockUser } from "../api/safety";
import { ActionSheet } from "../components/ActionSheet";
import { ConfirmModal } from "../components/ConfirmModal";
import { NavHeader } from "../components/NavHeader";
import { ProfileCard } from "../components/ProfileCard";
import { ReportModal } from "../components/ReportModal";
import { Screen } from "../components/Screen";
import { emitProfileSwiped } from "../hooks/deckBus";
import { queryKeys } from "../hooks/queries";
import type { MainStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { ErrorView } from "../shared/feedback";
import { Button, Loader, Text, spacing } from "../shared/ui";

type Props = NativeStackScreenProps<MainStackParamList, "Person">;
type Dialog = "menu" | "block" | "report" | "unmatch" | null;

export default function PersonScreen({ navigation, route }: Props) {
  const { userId, matchId, conversationId } = route.params;
  const client = useQueryClient();
  const [dialog, setDialog] = useState<Dialog>(null);

  const query = useQuery({ queryKey: queryKeys.person(userId), queryFn: () => getProfile(userId), retry: false });
  const profile = query.data;
  const gone = query.error instanceof ApiError && query.error.status === 404;

  const back = () => {
    void client.invalidateQueries({ queryKey: queryKeys.conversations });
    navigation.navigate("Tabs");
  };
  const fail = (error: unknown) => showToast(errorMessage(error), "error");

  const act = useMutation({
    mutationFn: (action: "like" | "pass") => swipe(userId, action),
    onSuccess: (result) => {
      emitProfileSwiped(userId, result);
      navigation.goBack();
    },
    onError: (error) => {
      if (error instanceof ApiError && error.status === 409) {
        navigation.goBack();
        return;
      }
      fail(error);
    }
  });
  const block = useMutation({
    mutationFn: () => blockUser(userId),
    onSuccess: () => {
      emitProfileSwiped(userId, { matched: false });
      showToast("Utilisateur bloqué.", "info");
      back();
    },
    onError: fail
  });
  const end = useMutation({
    mutationFn: async () => {
      if (!matchId) throw new Error("Match introuvable.");
      await unmatch(matchId);
    },
    onSuccess: () => {
      showToast("Match supprimé.", "info");
      back();
    },
    onError: fail
  });

  const name = profile?.firstName ?? "cette personne";

  return (
    <Screen
      header={
        <NavHeader
          onBack={() => navigation.goBack()}
          right={
            <Pressable accessibilityLabel="Plus d'options" accessibilityRole="button" hitSlop={10} onPress={() => setDialog("menu")} style={styles.more} testID="person-menu">
              <Text style={styles.moreGlyph} weight="bold">
                ⋯
              </Text>
            </Pressable>
          }
          title={profile?.firstName ?? "Profil"}
        />
      }
      scroll
      testID="person-screen"
    >
      {query.isLoading ? <Loader label="Chargement…" /> : null}
      {gone ? <ErrorView message="Ce profil n'est plus disponible." onAction={() => navigation.goBack()} actionLabel="Retour" title="Introuvable" /> : null}
      {query.isError && !gone ? <ErrorView actionLabel="Réessayer" message="Impossible de charger ce profil." onAction={() => void query.refetch()} title="Oups" /> : null}

      {profile ? (
        <>
          <View style={styles.card}>
            <ProfileCard profile={profile} variant="full" />
          </View>
          {conversationId || matchId ? (
            <Button
              label={`Écrire à ${profile.firstName}`}
              onPress={() => navigation.navigate("Conversation", { conversationId: conversationId ?? "", matchId, userId, name: profile.firstName })}
              disabled={!conversationId}
              size="lg"
              testID="person-message"
            />
          ) : (
            <View style={styles.row}>
              <Button label="Passer" loading={act.isPending && act.variables === "pass"} onPress={() => act.mutate("pass")} style={styles.flex} testID="person-pass" variant="outline" />
              <Button label="J'aime" loading={act.isPending && act.variables === "like"} onPress={() => act.mutate("like")} style={styles.flex} testID="person-like" />
            </View>
          )}
        </>
      ) : null}

      <ActionSheet
        actions={[
          ...(matchId ? [{ label: "Supprimer le match", onPress: () => setDialog("unmatch"), destructive: true, testID: "person-unmatch" }] : []),
          { label: "Bloquer", onPress: () => setDialog("block"), destructive: true, testID: "person-block" },
          { label: "Signaler", onPress: () => setDialog("report"), destructive: true, testID: "person-report" }
        ]}
        onClose={() => setDialog((d) => (d === "menu" ? null : d))}
        visible={dialog === "menu"}
      />
      <ConfirmModal
        confirmLabel="Bloquer"
        destructive
        loading={block.isPending}
        message={`${name} ne pourra plus vous voir ni vous écrire.`}
        onCancel={() => setDialog(null)}
        onConfirm={() => block.mutate()}
        title={`Bloquer ${name} ?`}
        visible={dialog === "block"}
      />
      <ConfirmModal
        confirmLabel="Supprimer le match"
        destructive
        loading={end.isPending}
        message="Vos messages seront supprimés pour vous deux."
        onCancel={() => setDialog(null)}
        onConfirm={() => end.mutate()}
        title="Supprimer le match ?"
        visible={dialog === "unmatch"}
      />
      <ReportModal firstName={name} onClose={() => setDialog(null)} onDone={() => { emitProfileSwiped(userId, { matched: false }); back(); }} userId={userId} visible={dialog === "report"} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  card: { height: 560 },
  row: { flexDirection: "row", gap: spacing.md },
  flex: { flex: 1 },
  more: { width: 44, height: 44, alignItems: "center", justifyContent: "center" },
  moreGlyph: { fontSize: 26, lineHeight: 30 }
});
