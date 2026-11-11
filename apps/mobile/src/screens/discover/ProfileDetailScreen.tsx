import React from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { discoveryApi, safetyApi } from "../../api/endpoints";
import type { SwipeDirection } from "../../api/types";
import { keys, useProfileCard, useSwipe } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Chip, ErrorState, IconButton, LoadingState, Photo, Text, useFeedback } from "../../ui";
import { formatDistance, genderLabels } from "../../utils/format";

type Props = NativeStackScreenProps<AppStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ route, navigation }: Props) {
  const { userId, card: initial, matchId } = route.params;
  const { colors, spacing, radii } = useTheme();
  const { width } = useWindowDimensions();
  const qc = useQueryClient();
  const { toast, confirm } = useFeedback();
  const profile = useProfileCard(userId, initial);
  const swipe = useSwipe();

  const block = useMutation({
    mutationFn: () => safetyApi.block(userId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.conversations });
      void qc.invalidateQueries({ queryKey: keys.discover });
      toast("Profil bloqué. Vous ne vous verrez plus.", "success");
      navigation.popToTop();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });
  const unmatch = useMutation({
    mutationFn: () => discoveryApi.unmatch(matchId ?? ""),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.conversations });
      toast("Match annulé.");
      navigation.popToTop();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  if (profile.isLoading) {
    return <LoadingState />;
  }
  if (profile.isError || !profile.data) {
    return <ErrorState message="Ce profil n'est plus disponible." onRetry={() => navigation.goBack()} />;
  }
  const card = profile.data;
  const photoWidth = Math.min(width, 640);
  const distance = formatDistance(card.distanceKm);

  const decide = (direction: SwipeDirection) => {
    swipe.mutate(
      { card, direction },
      {
        onSuccess: (res) => {
          if (res.matched && res.match) {
            navigation.replace("Chat", { matchId: res.match.matchId, user: res.match.user });
            toast(`C'est un match avec ${card.firstName} !`, "success");
          } else {
            navigation.goBack();
          }
        },
        onError: (e) => toast(errorMessage(e), "error")
      }
    );
  };

  return (
    <SafeAreaView
      edges={["bottom"]}
      style={[styles.flex, { backgroundColor: colors.background }]}
      testID="profile-detail"
    >
      <ScrollView contentContainerStyle={{ paddingBottom: 120 }}>
        <View style={[styles.column, { width: photoWidth }]}>
          {card.photos.map((p, i) => (
            <Photo
              key={p.id}
              url={p.url}
              style={{ width: photoWidth, height: photoWidth * 1.25, marginBottom: i === 0 ? 0 : 4 }}
              accessibilityLabel={`Photo ${i + 1} de ${card.firstName}`}
            />
          ))}
          <View style={{ padding: spacing.lg, gap: spacing.md }}>
            <Text variant="display">
              {card.firstName}, {card.age}
            </Text>
            <View style={{ gap: 4 }}>
              {card.jobTitle ? <Text tone="muted">{card.jobTitle}</Text> : null}
              <Text tone="muted">{[genderLabels[card.gender], distance, card.city].filter(Boolean).join(" · ")}</Text>
            </View>
            {card.bio ? (
              <View
                style={[
                  styles.block,
                  { backgroundColor: colors.surface, borderColor: colors.border, borderRadius: radii.lg }
                ]}
              >
                <Text variant="overline" tone="subtle">
                  À propos
                </Text>
                <Text>{card.bio}</Text>
              </View>
            ) : null}
            {card.interests.length > 0 ? (
              <View style={{ gap: 8 }}>
                <Text variant="overline" tone="subtle">
                  Centres d’intérêt
                </Text>
                <View style={styles.chips}>
                  {card.interests.map((i) => (
                    <Chip key={i.slug} label={i.label} highlighted={i.shared} />
                  ))}
                </View>
                {card.interests.some((i) => i.shared) ? (
                  <Text variant="caption" tone="primary">
                    En couleur : vos points communs
                  </Text>
                ) : null}
              </View>
            ) : null}
            <View style={{ gap: 8, marginTop: spacing.md }}>
              {matchId ? (
                <Button
                  label="Annuler le match"
                  variant="secondary"
                  icon="heart-dislike-outline"
                  loading={unmatch.isPending}
                  onPress={async () => {
                    if (
                      await confirm({
                        title: `Annuler le match avec ${card.firstName} ?`,
                        message: "La conversation sera supprimée pour vous deux.",
                        confirmLabel: "Annuler le match",
                        destructive: true
                      })
                    ) {
                      unmatch.mutate();
                    }
                  }}
                />
              ) : null}
              <Button
                testID="profile-block"
                label={`Bloquer ${card.firstName}`}
                variant="secondary"
                icon="ban-outline"
                loading={block.isPending}
                onPress={async () => {
                  if (
                    await confirm({
                      title: `Bloquer ${card.firstName} ?`,
                      message: "Vous ne pourrez plus vous voir ni vous écrire. Votre match éventuel sera supprimé.",
                      confirmLabel: "Bloquer",
                      destructive: true
                    })
                  ) {
                    block.mutate();
                  }
                }}
              />
              <Button
                testID="profile-report"
                label="Signaler ce profil"
                variant="danger"
                icon="flag-outline"
                onPress={() => navigation.navigate("Report", { userId, firstName: card.firstName })}
              />
            </View>
          </View>
        </View>
      </ScrollView>
      {!matchId ? (
        <View style={[styles.floating, { gap: spacing.xl }]} pointerEvents="box-none">
          <IconButton
            testID="detail-pass"
            icon="close"
            label="Passer"
            size={60}
            filled
            onPress={() => decide("pass")}
            disabled={swipe.isPending}
          />
          <IconButton
            testID="detail-like"
            icon="heart"
            label="J'aime"
            size={68}
            filled
            tone="primary"
            onPress={() => decide("like")}
            disabled={swipe.isPending}
          />
        </View>
      ) : null}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  column: { alignSelf: "center" },
  block: { padding: 16, gap: 8, borderWidth: 1 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  floating: { position: "absolute", bottom: 24, left: 0, right: 0, flexDirection: "row", justifyContent: "center" }
});
