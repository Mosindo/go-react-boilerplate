import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import React, { useLayoutEffect } from "react";
import { Pressable, StyleSheet, View } from "react-native";

import { discoverApi } from "../../api/endpoints";
import { useSafetyActions } from "../../lib/useSafetyActions";
import type { MainStackParamList } from "../../navigation/types";
import { keys } from "../../realtime/RealtimeProvider";
import { spacing, useTheme } from "../../theme";
import { Button } from "../../ui/Button";
import { useFeedback } from "../../ui/Feedback";
import { PhotoCarousel } from "../../ui/PhotoCarousel";
import { ProfileBody } from "../../ui/ProfileBody";
import { Screen } from "../../ui/Screen";
import { ErrorState, Loading, errorMessage } from "../../ui/States";

type Props = NativeStackScreenProps<MainStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ navigation, route }: Props) {
  const { colors } = useTheme();
  const qc = useQueryClient();
  const { toast, sheet } = useFeedback();
  const { block, report } = useSafetyActions();
  const { userId } = route.params;

  const query = useQuery({
    queryKey: ["profileDetail", userId],
    queryFn: () => discoverApi.profile(userId),
  });
  const card = query.data;

  const matches = qc.getQueryData<{ pages: { matches: { id: string; user: { id: string } }[] }[] }>(
    keys.matches,
  );
  const existingMatch = matches?.pages.flatMap((p) => p.matches).find((m) => m.user.id === userId);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: card?.firstName ?? "",
      headerRight: () =>
        card ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Plus d'options"
            hitSlop={12}
            onPress={() =>
              sheet(card.firstName, [
                {
                  label: "Signaler",
                  onPress: () => report(card.id, card.firstName, () => navigation.popToTop()),
                },
                {
                  label: "Bloquer",
                  destructive: true,
                  onPress: () => void block(card.id, card.firstName, () => navigation.popToTop()),
                },
              ])
            }
          >
            <Ionicons name="ellipsis-horizontal" size={24} color={colors.text} />
          </Pressable>
        ) : null,
    });
  }, [navigation, card, sheet, report, block, colors.text]);

  const act = async (action: "like" | "pass") => {
    try {
      const res = await discoverApi.swipe(userId, action);
      void qc.invalidateQueries({ queryKey: keys.discover });
      if (res.matched && res.matchId && card) {
        void qc.invalidateQueries({ queryKey: keys.matches });
        navigation.replace("Chat", { matchId: res.matchId, name: card.firstName, userId });
        toast("C'est un match !", "success");
      } else {
        navigation.goBack();
      }
    } catch (err) {
      toast(errorMessage(err), "error");
    }
  };

  if (query.isLoading) return <Loading />;
  if (query.isError || !card)
    return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;

  return (
    <Screen scroll edges={["bottom"]}>
      <PhotoCarousel photos={card.photos} name={card.firstName} />
      <ProfileBody card={card} />
      {existingMatch ? (
        <Button
          label="Envoyer un message"
          onPress={() =>
            navigation.navigate("Chat", { matchId: existingMatch.id, name: card.firstName, userId })
          }
        />
      ) : (
        <View style={styles.row}>
          <Button
            label="Passer"
            variant="secondary"
            style={styles.flex}
            onPress={() => void act("pass")}
          />
          <Button label="J'aime" style={styles.flex} onPress={() => void act("like")} />
        </View>
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", gap: spacing.md },
  flex: { flex: 1 },
});
