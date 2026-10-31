import React, { useState } from "react";
import { Alert, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { blockUser, getProfile, swipe } from "../api/dating";
import { ProfileCardView } from "../components/ProfileCardView";
import { ReportSheet } from "../components/ReportSheet";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParamList } from "../navigation/types";
import { Screen } from "../shared/layout";
import { ErrorView, LoadingView } from "../shared/feedback";
import { Button, Notice, spacing } from "../shared/ui";

type Props = NativeStackScreenProps<MainStackParamList, "ProfileDetail">;

export default function ProfileDetailScreen({ navigation, route }: Props) {
  const { id, swipeable } = route.params;
  const queryClient = useQueryClient();
  const [reporting, setReporting] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  const card = useQuery({ queryKey: queryKeys.card(id), queryFn: () => getProfile(id) });

  const refreshLists = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.feed });
    void queryClient.invalidateQueries({ queryKey: queryKeys.received });
    void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
    void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
  };

  const decide = useMutation({
    mutationFn: (kind: "like" | "pass") => swipe(id, kind),
    onSuccess: (result) => {
      refreshLists();
      if (result.matched && result.match) {
        navigation.replace("Chat", { conversationId: result.match.conversationId });
      } else {
        navigation.goBack();
      }
    },
    onError: () => setNotice("Action impossible pour le moment.")
  });

  const block = useMutation({
    mutationFn: () => blockUser(id),
    onSuccess: () => {
      refreshLists();
      navigation.popToTop();
    }
  });

  if (card.isLoading)
    return (
      <Screen>
        <LoadingView label="Chargement du profil…" />
      </Screen>
    );
  if (card.isError || !card.data) {
    return (
      <Screen>
        <ErrorView actionLabel="Retour" message="Ce profil n'est plus disponible." onAction={() => navigation.goBack()} />
      </Screen>
    );
  }
  const profile = card.data;

  return (
    <Screen scroll>
      <ProfileCardView card={profile} detailed />
      {notice ? <Notice title={notice} tone="warning" /> : null}
      {swipeable ? (
        <View style={styles.row}>
          <Button label="Passer" loading={decide.isPending} onPress={() => decide.mutate("pass")} style={styles.flex} variant="outline" />
          <Button label="J'aime" loading={decide.isPending} onPress={() => decide.mutate("like")} style={styles.flex} />
        </View>
      ) : null}
      <View style={styles.row}>
        <Button label="Signaler" onPress={() => setReporting(true)} style={styles.flex} variant="ghost" />
        <Button
          label="Bloquer"
          onPress={() =>
            Alert.alert(`Bloquer ${profile.firstName} ?`, "Cette personne ne pourra plus vous voir ni vous écrire.", [
              { text: "Annuler", style: "cancel" },
              { text: "Bloquer", style: "destructive", onPress: () => block.mutate() }
            ])
          }
          style={styles.flex}
          variant="ghost"
        />
      </View>
      <ReportSheet
        name={profile.firstName}
        onClose={(reported) => {
          setReporting(false);
          if (reported) {
            refreshLists();
            navigation.popToTop();
          }
        }}
        userId={reporting ? profile.id : null}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", gap: spacing.md },
  flex: { flex: 1 }
});
