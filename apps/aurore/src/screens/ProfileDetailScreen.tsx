import React from "react";
import { ScrollView, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { discoverApi, moderationApi } from "../api/endpoints";
import { keys } from "../api/keys";
import { useFeedback } from "../components/Feedback";
import { errorMessage } from "../components/forms";
import { PhotoCarousel, ProfileBody } from "../components/profile";
import { Button, ErrorState, Loading, Screen } from "../components/ui";
import type { RootStackParamList } from "../navigation/types";
import { spacing } from "../theme/theme";

type Props = NativeStackScreenProps<RootStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, matchId, conversationId } = route.params;
  const qc = useQueryClient();
  const { toast, confirm } = useFeedback();
  const card = useQuery({ queryKey: keys.cardOf(userId), queryFn: () => discoverApi.profile(userId) });
  const matches = useQuery({ queryKey: keys.matches, queryFn: discoverApi.matches, enabled: !!conversationId && !matchId });
  const resolvedMatchId = matchId ?? matches.data?.find((m) => m.user.userId === userId)?.id;

  const leave = () => {
    void qc.invalidateQueries({ queryKey: keys.matches });
    void qc.invalidateQueries({ queryKey: keys.conversations });
    void qc.invalidateQueries({ queryKey: ["discover"] });
    navigation.popToTop();
  };

  const block = useMutation({
    mutationFn: () => moderationApi.block(userId),
    onSuccess: () => {
      toast("Personne bloquée.", "success");
      leave();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });
  const unmatch = useMutation({
    mutationFn: () => discoverApi.unmatch(resolvedMatchId ?? ""),
    onSuccess: () => {
      toast("Match supprimé.", "success");
      leave();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  if (card.isLoading) return <Screen><Loading /></Screen>;
  if (card.error || !card.data) {
    return <Screen><ErrorState message="Ce profil n'est plus disponible." onRetry={() => void card.refetch()} /></Screen>;
  }
  const c = card.data;

  const askBlock = async () => {
    const ok = await confirm({
      title: `Bloquer ${c.firstName} ?`,
      message: "Vous ne vous verrez plus et le match éventuel sera supprimé. Vous pourrez débloquer depuis les réglages.",
      confirmLabel: "Bloquer",
      destructive: true
    });
    if (ok) block.mutate();
  };
  const askUnmatch = async () => {
    const ok = await confirm({
      title: `Supprimer le match avec ${c.firstName} ?`,
      message: "La conversation sera supprimée pour vous deux.",
      confirmLabel: "Supprimer le match",
      destructive: true
    });
    if (ok) unmatch.mutate();
  };

  return (
    <Screen edges={["left", "right", "bottom"]} padded={false}>
      <ScrollView>
        <PhotoCarousel card={c} />
        <ProfileBody card={c} />
        <View style={{ padding: spacing.lg, gap: spacing.md }}>
          {resolvedMatchId ? <Button label="Supprimer le match" variant="secondary" onPress={() => void askUnmatch()} loading={unmatch.isPending} testID="profile-unmatch" /> : null}
          <Button label="Signaler" variant="secondary" icon="flag" onPress={() => navigation.navigate("Report", { userId, name: c.firstName })} testID="profile-report" />
          <Button label="Bloquer" variant="danger" icon="ban" onPress={() => void askBlock()} loading={block.isPending} testID="profile-block" />
        </View>
      </ScrollView>
    </Screen>
  );
}
