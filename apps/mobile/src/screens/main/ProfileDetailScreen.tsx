import React, { useLayoutEffect, useState } from "react";
import { Alert, FlatList, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { getProfile } from "../../api/profile";
import { blockUser, swipe, unmatch } from "../../api/social";
import type { Match } from "../../api/types";
import { MatchModal } from "../../components/MatchModal";
import { distanceLabel } from "../../components/SwipeCard";
import { errorMessage } from "../../lib/errors";
import type { MainStackParams } from "../../navigation/types";
import { Chip, ErrorState, Loading, PhotoImage, Screen, Text, useActionSheet, useToast } from "../../ui";
import { MIN_TOUCH, radius, spacing } from "../../theme/tokens";
import { useTheme } from "../../theme/ThemeProvider";

type Props = NativeStackScreenProps<MainStackParams, "ProfileDetail">;

export default function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, profile: initial, canSwipe, matchId } = route.params;
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const toast = useToast();
  const sheet = useActionSheet();
  const queryClient = useQueryClient();
  const [photoIndex, setPhotoIndex] = useState(0);
  const [match, setMatch] = useState<Match | null>(null);
  const [busy, setBusy] = useState(false);
  const query = useQuery({
    queryKey: ["profile", userId],
    queryFn: () => getProfile(userId),
    ...(initial ? { initialData: initial } : {})
  });
  const profile = query.data;

  const doBlock = () => {
    if (!profile) return;
    Alert.alert(
      `Bloquer ${profile.firstName} ?`,
      "Vous ne vous verrez plus mutuellement et votre conversation éventuelle sera supprimée.",
      [
        { text: "Annuler", style: "cancel" },
        {
          text: "Bloquer",
          style: "destructive",
          onPress: () => {
            blockUser(userId)
              .then(() => {
                void queryClient.invalidateQueries();
                navigation.popToTop();
                toast.show(`${profile.firstName} a été bloqué(e).`, "success");
              })
              .catch((e) => toast.show(errorMessage(e), "error"));
          }
        }
      ]
    );
  };

  const doUnmatch = () => {
    if (!matchId || !profile) return;
    Alert.alert(
      `Supprimer le match avec ${profile.firstName} ?`,
      "La conversation sera supprimée pour vous deux.",
      [
        { text: "Annuler", style: "cancel" },
        {
          text: "Supprimer",
          style: "destructive",
          onPress: () => {
            unmatch(matchId)
              .then(() => {
                void queryClient.invalidateQueries({ queryKey: ["conversations"] });
                navigation.popToTop();
              })
              .catch((e) => toast.show(errorMessage(e), "error"));
          }
        }
      ]
    );
  };

  useLayoutEffect(() => {
    navigation.setOptions({
      title: "",
      headerRight: () => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Plus d'options"
          style={styles.menu}
          onPress={() =>
            sheet.show({
              title: profile?.firstName ?? "Profil",
              actions: [
                {
                  label: "Signaler",
                  onPress: () => profile && navigation.navigate("Report", { userId, name: profile.firstName })
                },
                { label: "Bloquer", destructive: true, onPress: doBlock },
                ...(matchId ? [{ label: "Supprimer le match", destructive: true, onPress: doUnmatch }] : [])
              ]
            })
          }
          testID="profile-menu"
        >
          <Text variant="title">⋯</Text>
        </Pressable>
      )
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [navigation, profile, matchId]);

  if (query.isPending) return <Loading />;
  if (query.isError || !profile)
    return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;

  const decide = async (action: "like" | "pass") => {
    setBusy(true);
    try {
      const result = await swipe(userId, action);
      void queryClient.invalidateQueries({ queryKey: ["likes"] });
      if (result.matched && result.match && !result.duplicate) setMatch(result.match);
      else navigation.goBack();
    } catch (e) {
      toast.show(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  const distance = distanceLabel(profile.distanceKm);
  return (
    <Screen scroll padded={false} edges={["bottom"]} testID="profile-detail">
      <FlatList
        horizontal
        pagingEnabled
        showsHorizontalScrollIndicator={false}
        data={profile.photos}
        keyExtractor={(p) => p.id}
        onMomentumScrollEnd={(e) => setPhotoIndex(Math.round(e.nativeEvent.contentOffset.x / width))}
        renderItem={({ item }) => (
          <PhotoImage photoId={item.id} name={profile.firstName} style={{ width, height: width * 1.25 }} />
        )}
        ListEmptyComponent={<PhotoImage name={profile.firstName} style={{ width, height: width * 1.25 }} />}
      />
      {profile.photos.length > 1 ? (
        <Text variant="caption" tone="muted" style={styles.counter}>
          {photoIndex + 1} / {profile.photos.length}
        </Text>
      ) : null}
      <View style={styles.body}>
        <Text variant="display">
          {profile.firstName}, {profile.age}
        </Text>
        {profile.city || distance ? (
          <Text tone="muted">{[profile.city, distance].filter(Boolean).join(" · ")}</Text>
        ) : null}
        {profile.bio ? <Text>{profile.bio}</Text> : null}
        {profile.interests.length > 0 ? (
          <View style={styles.chips}>
            {profile.interests.map((i) => (
              <Chip key={i.id} label={i.label} />
            ))}
          </View>
        ) : null}
      </View>
      {canSwipe ? (
        <View style={[styles.bar, { borderColor: colors.border, backgroundColor: colors.surface }]}>
          <ActionButton
            label="Passer"
            symbol="✕"
            color={colors.pass}
            disabled={busy}
            onPress={() => void decide("pass")}
            testID="detail-pass"
          />
          <ActionButton
            label="J'aime"
            symbol="♥"
            color={colors.like}
            disabled={busy}
            onPress={() => void decide("like")}
            testID="detail-like"
          />
        </View>
      ) : null}
      <MatchModal
        match={match}
        onClose={() => {
          setMatch(null);
          navigation.goBack();
        }}
        onMessage={(m) => {
          setMatch(null);
          navigation.replace("Chat", { conversationId: m.conversationId, name: m.profile.firstName });
        }}
      />
    </Screen>
  );
}

function ActionButton({
  label,
  symbol,
  color,
  disabled,
  onPress,
  testID
}: {
  label: string;
  symbol: string;
  color: string;
  disabled: boolean;
  onPress: () => void;
  testID: string;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      disabled={disabled}
      onPress={onPress}
      testID={testID}
      style={[styles.action, { borderColor: color, opacity: disabled ? 0.5 : 1 }]}
    >
      <Text style={{ color }} bold>
        {symbol} {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  menu: { minWidth: MIN_TOUCH, minHeight: MIN_TOUCH, alignItems: "center", justifyContent: "center" },
  counter: { textAlign: "center", paddingTop: spacing.sm },
  body: { padding: spacing.lg, gap: spacing.md },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  bar: {
    flexDirection: "row",
    gap: spacing.md,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  action: {
    flex: 1,
    minHeight: MIN_TOUCH + 6,
    borderWidth: 2,
    borderRadius: radius.pill,
    alignItems: "center",
    justifyContent: "center"
  }
});
