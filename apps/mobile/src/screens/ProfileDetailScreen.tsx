import { useCallback, useLayoutEffect, useState } from "react";
import { Alert, FlatList, Pressable, ScrollView, View, useWindowDimensions } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { discoverApi, moderationApi, profileApi } from "../api";
import { useToast } from "../components/feedback";
import { Button, Chip, ErrorState, Loading, PhotoImage, Text } from "../components/ui";
import { ApiError, errorMessage } from "../lib/errors";
import { formatDistance } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { keys } from "../state/queries";
import { spacing, useColors } from "../theme";

type Props = NativeStackScreenProps<RootStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, canSwipe } = route.params;
  const colors = useColors();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { width } = useWindowDimensions();
  const [index, setIndex] = useState(0);
  const [busy, setBusy] = useState(false);

  const profile = useQuery({ queryKey: keys.publicProfile(userId), queryFn: () => profileApi.getPublic(userId) });
  const data = profile.data;

  const moderate = useCallback(() => {
    if (!data) {
      return;
    }
    Alert.alert(data.firstName, undefined, [
      { text: "Signaler", onPress: () => navigation.navigate("Report", { userId, name: data.firstName }) },
      {
        text: "Bloquer",
        style: "destructive",
        onPress: () =>
          Alert.alert(`Bloquer ${data.firstName} ?`, "Cette personne ne pourra plus vous voir ni vous écrire.", [
            { text: "Annuler", style: "cancel" },
            {
              text: "Bloquer",
              style: "destructive",
              onPress: () =>
                moderationApi
                  .block(userId)
                  .then(() => {
                    void queryClient.invalidateQueries({ queryKey: keys.matches });
                    void queryClient.invalidateQueries({ queryKey: keys.conversations });
                    toast.show(`${data.firstName} est bloqué(e).`, "success");
                    navigation.popToTop();
                  })
                  .catch((e) => toast.show(errorMessage(e), "error")),
            },
          ]),
      },
      { text: "Fermer", style: "cancel" },
    ]);
  }, [data, navigation, queryClient, toast, userId]);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: data?.firstName ?? "Profil",
      headerRight: () => (
        <Pressable accessibilityLabel="Options" accessibilityRole="button" hitSlop={10} onPress={moderate}>
          <Ionicons color={colors.text} name="ellipsis-horizontal" size={24} />
        </Pressable>
      ),
    });
  }, [navigation, data?.firstName, moderate, colors.text]);

  const swipe = async (action: "like" | "pass") => {
    setBusy(true);
    try {
      const result = action === "like" ? await discoverApi.like(userId) : await discoverApi.pass(userId);
      void queryClient.invalidateQueries({ queryKey: keys.matches });
      navigation.goBack();
      if (result.matched && result.matchId && result.conversationId && data) {
        navigation.navigate("MatchModal", {
          matchId: result.matchId,
          conversationId: result.conversationId,
          userId,
          firstName: data.firstName,
          photoId: data.photoIds[0],
        });
      }
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        navigation.goBack();
      } else {
        toast.show(errorMessage(e), "error");
      }
    } finally {
      setBusy(false);
    }
  };

  if (profile.isLoading) {
    return <Loading />;
  }
  if (profile.isError || !data) {
    return <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;
  }
  const distance = formatDistance(data.distanceKm);

  return (
    <View style={{ flex: 1, backgroundColor: colors.background }}>
      <ScrollView>
        <FlatList
          data={data.photoIds}
          horizontal
          keyExtractor={(id) => id}
          onMomentumScrollEnd={(e) => setIndex(Math.round(e.nativeEvent.contentOffset.x / width))}
          pagingEnabled
          renderItem={({ item }) => <PhotoImage photoId={item} style={{ width, height: width * 1.25 }} />}
          showsHorizontalScrollIndicator={false}
        />
        {data.photoIds.length > 1 ? (
          <Text muted style={{ textAlign: "center", marginTop: spacing.sm }} variant="caption">
            {index + 1} / {data.photoIds.length}
          </Text>
        ) : null}
        <View style={{ padding: spacing.lg }}>
          <Text variant="title">
            {data.firstName}, {data.age}
          </Text>
          {data.city || distance ? (
            <Text muted style={{ marginTop: spacing.xs }}>
              {[data.city, distance].filter(Boolean).join(" · ")}
            </Text>
          ) : null}
          {data.bio ? <Text style={{ marginTop: spacing.lg }}>{data.bio}</Text> : null}
          {data.interests.length > 0 ? (
            <View style={{ marginTop: spacing.lg }}>
              <Text style={{ marginBottom: spacing.sm }} variant="heading">
                Centres d'intérêt
              </Text>
              <View style={{ flexDirection: "row", flexWrap: "wrap" }}>
                {data.interests.map((interest) => (
                  <Chip key={interest.slug} label={interest.label} />
                ))}
              </View>
            </View>
          ) : null}
        </View>
      </ScrollView>
      {canSwipe ? (
        <View style={{ flexDirection: "row", gap: spacing.md, padding: spacing.lg, backgroundColor: colors.surface }}>
          <Button
            disabled={busy}
            onPress={() => void swipe("pass")}
            style={{ flex: 1 }}
            title="Passer"
            variant="secondary"
          />
          <Button disabled={busy} icon="heart" onPress={() => void swipe("like")} style={{ flex: 1 }} title="J'aime" />
        </View>
      ) : null}
    </View>
  );
}
