import React, { useLayoutEffect, useRef, useState } from "react";
import { Alert, Dimensions, FlatList, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { getProfileOf, swipe, unmatch } from "../api/dating";
import { blockUser } from "../api/platform";
import type { SwipeAction } from "../api/types";
import { MatchModal } from "../components/MatchModal";
import { PhotoImage } from "../components/PhotoImage";
import { useAuth } from "../hooks/useAuth";
import { describeError } from "../lib/errors";
import { formatDistance, genderLabel, nameWithAge } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Badge, Button, Text, spacing, useTheme } from "../shared/ui";
import { emitSwiped } from "../state/swipeBus";

type Props = NativeStackScreenProps<RootStackParamList, "ProfileDetail">;

const WIDTH = Dimensions.get("window").width;

export default function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, context, matchId } = route.params;
  const { colors } = useTheme();
  const { self } = useAuth();
  const queryClient = useQueryClient();
  const [photoIndex, setPhotoIndex] = useState(0);
  const [busy, setBusy] = useState(false);
  const [matched, setMatched] = useState<string | null>(null);
  const listRef = useRef<FlatList>(null);

  const query = useQuery({ queryKey: ["profile", userId], queryFn: () => getProfileOf(userId), retry: false });
  const profile = query.data;

  const leaveAfter = async (message: string) => {
    await queryClient.invalidateQueries({ queryKey: ["matches"] });
    await queryClient.invalidateQueries({ queryKey: ["likes"] });
    showToast(message, "success");
    navigation.popToTop();
  };

  const confirmBlock = () =>
    Alert.alert(
      `Bloquer ${profile?.firstName ?? "cette personne"} ?`,
      "Vous ne vous verrez plus et ne pourrez plus vous écrire. Vous pouvez débloquer à tout moment dans les paramètres.",
      [
        { text: "Annuler", style: "cancel" },
        {
          text: "Bloquer",
          style: "destructive",
          onPress: () => {
            void blockUser(userId)
              .then(() => leaveAfter("Personne bloquée"))
              .catch((err) => showToast(describeError(err), "danger"));
          }
        }
      ]
    );

  const confirmUnmatch = () => {
    if (!matchId) {
      return;
    }
    Alert.alert("Défaire le match ?", "La conversation sera fermée pour vous deux.", [
      { text: "Annuler", style: "cancel" },
      {
        text: "Défaire",
        style: "destructive",
        onPress: () => {
          void unmatch(matchId)
            .then(() => leaveAfter("Match défait"))
            .catch((err) => showToast(describeError(err), "danger"));
        }
      }
    ]);
  };

  useLayoutEffect(() => {
    navigation.setOptions({
      title: profile?.firstName ?? "",
      headerRight: () => (
        <Pressable
          accessibilityLabel="Plus d'options"
          accessibilityRole="button"
          disabled={!profile}
          hitSlop={12}
          onPress={() =>
            Alert.alert(profile?.firstName ?? "Profil", undefined, [
              ...(context === "match" && matchId ? [{ text: "Défaire le match", onPress: confirmUnmatch }] : []),
              { text: "Signaler", onPress: () => navigation.navigate("Report", { userId, name: profile?.firstName ?? "" }) },
              { text: "Bloquer", style: "destructive" as const, onPress: confirmBlock },
              { text: "Annuler", style: "cancel" as const }
            ])
          }
          testID="profile-menu"
        >
          <Ionicons color={colors.text} name="ellipsis-horizontal" size={24} />
        </Pressable>
      )
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [navigation, profile, colors.text, context, matchId]);

  const decide = async (action: SwipeAction) => {
    if (!profile || busy) {
      return;
    }
    setBusy(true);
    try {
      const result = await swipe(userId, action);
      emitSwiped(userId, action);
      void queryClient.invalidateQueries({ queryKey: ["likes"] });
      if (result.matched && !result.alreadyProcessed && result.matchId) {
        void queryClient.invalidateQueries({ queryKey: ["matches"] });
        setMatched(result.matchId);
      } else {
        navigation.goBack();
      }
    } catch (err) {
      showToast(describeError(err, "Action impossible. Réessayez."), "danger");
    } finally {
      setBusy(false);
    }
  };

  if (query.isLoading) {
    return <LoadingView fullScreen label="Chargement du profil…" />;
  }
  if (query.isError || !profile) {
    const gone = (query.error as { status?: number } | null)?.status === 404;
    return (
      <ErrorView
        message={gone ? "Ce profil n'est plus disponible." : describeError(query.error)}
        onAction={gone ? () => navigation.goBack() : () => void query.refetch()}
        actionLabel={gone ? "Retour" : "Réessayer"}
      />
    );
  }

  const where = [profile.city, formatDistance(profile.distanceKm)].filter(Boolean).join(" · ");
  const photos = profile.photos;

  return (
    <View style={[styles.flex, { backgroundColor: colors.background }]} testID="profile-detail">
      <ScrollView contentContainerStyle={styles.content}>
        <View style={styles.carousel}>
          <FlatList
            data={photos}
            horizontal
            keyExtractor={(photo) => photo.id}
            onMomentumScrollEnd={(event) => setPhotoIndex(Math.round(event.nativeEvent.contentOffset.x / WIDTH))}
            pagingEnabled
            ref={listRef}
            renderItem={({ item, index }) => (
              <PhotoImage
                accessibilityLabel={`Photo ${index + 1} sur ${photos.length}`}
                path={item.url}
                style={{ width: WIDTH, height: WIDTH * 1.25 }}
              />
            )}
            showsHorizontalScrollIndicator={false}
          />
          {photos.length > 1 ? (
            <View pointerEvents="none" style={styles.dots}>
              {photos.map((photo, index) => (
                <View
                  key={photo.id}
                  style={[styles.dotBar, { backgroundColor: index === photoIndex ? "#FFFFFF" : "rgba(255,255,255,0.45)" }]}
                />
              ))}
            </View>
          ) : null}
        </View>

        <View style={styles.body}>
          <View style={styles.nameBlock}>
            <Text variant="title">{nameWithAge(profile.firstName, profile.age)}</Text>
            <Text tone="muted">{[genderLabel(profile.gender), where].filter(Boolean).join(" · ")}</Text>
          </View>

          {profile.bio ? (
            <View style={styles.block}>
              <Text tone="secondary" variant="eyebrow" weight="bold">
                À propos
              </Text>
              <Text>{profile.bio}</Text>
            </View>
          ) : null}

          {profile.interests.length > 0 ? (
            <View style={styles.block}>
              <Text tone="secondary" variant="eyebrow" weight="bold">
                Centres d'intérêt
              </Text>
              <View style={styles.chips}>
                {profile.interests.map((interest) => (
                  <Badge key={interest.id} label={interest.label} tone={interest.shared ? "primary" : "neutral"} />
                ))}
              </View>
              {profile.interests.some((interest) => interest.shared) ? (
                <Text tone="muted" variant="caption">
                  En couleur : ce que vous avez en commun.
                </Text>
              ) : null}
            </View>
          ) : null}
        </View>
      </ScrollView>

      {context === "match" && matchId ? (
        <View style={[styles.bar, { backgroundColor: colors.surface, borderTopColor: colors.border }]}>
          <Button
            label={`Écrire à ${profile.firstName}`}
            onPress={() => navigation.navigate("Chat", { matchId, peerId: userId, peerName: profile.firstName })}
          />
        </View>
      ) : (
        <View style={[styles.bar, styles.barRow, { backgroundColor: colors.surface, borderTopColor: colors.border }]}>
          <Button
            disabled={busy}
            label="Passer"
            onPress={() => void decide("pass")}
            style={styles.half}
            testID="detail-pass"
            variant="outline"
          />
          <Button label="J'aime" loading={busy} onPress={() => void decide("like")} style={styles.half} testID="detail-like" />
        </View>
      )}

      <MatchModal
        myName={self?.profile?.firstName ?? ""}
        myPhotoPath={self?.photos[0]?.url}
        onClose={() => {
          setMatched(null);
          navigation.goBack();
        }}
        onMessage={() => {
          const id = matched;
          setMatched(null);
          if (id) {
            navigation.replace("Chat", { matchId: id, peerId: userId, peerName: profile.firstName });
          }
        }}
        otherName={profile.firstName}
        otherPhotoPath={photos[0]?.url}
        visible={matched !== null}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { paddingBottom: spacing.xl },
  carousel: { width: WIDTH, height: WIDTH * 1.25 },
  dots: { position: "absolute", top: spacing.sm, left: spacing.md, right: spacing.md, flexDirection: "row", gap: 4 },
  dotBar: { flex: 1, height: 3, borderRadius: 2 },
  body: { padding: spacing.xl, gap: spacing.xl },
  nameBlock: { gap: spacing.xxs },
  block: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  bar: { padding: spacing.lg, borderTopWidth: StyleSheet.hairlineWidth },
  barRow: { flexDirection: "row", gap: spacing.md },
  half: { flex: 1 }
});
