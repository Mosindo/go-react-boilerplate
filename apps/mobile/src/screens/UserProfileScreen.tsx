import React, { useState } from "react";
import { FlatList, StyleSheet, useWindowDimensions, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { discoveryApi } from "../api/endpoints";
import { GENDER_LABELS } from "../api/types";
import { useInterests } from "../features/useInterests";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParams } from "../navigation/types";
import { useRealtime } from "../realtime/RealtimeProvider";
import { useTheme } from "../theme";
import { Button, Chip, ErrorState, LoadingState, PhotoImage, Screen, Text, useToast } from "../ui";
import { SafetyMenu } from "../features/SafetyMenu";

type Props = NativeStackScreenProps<MainStackParams, "UserProfile">;

export function UserProfileScreen({ route, navigation }: Props) {
  const { userId, canSwipe } = route.params;
  const t = useTheme();
  const { width } = useWindowDimensions();
  const toast = useToast();
  const qc = useQueryClient();
  const { showMatch } = useRealtime();
  const interests = useInterests();
  const [index, setIndex] = useState(0);
  const query = useQuery({
    queryKey: queryKeys.profile(userId),
    queryFn: () => discoveryApi.profile(userId)
  });
  const labels = new Map((interests.data ?? []).map((i) => [i.slug, i.label]));

  const swipe = useMutation({
    mutationFn: (action: "like" | "pass") => discoveryApi.swipe(userId, action),
    onSuccess: (res) => {
      qc.removeQueries({ queryKey: queryKeys.feed });
      const name = query.data?.firstName ?? "";
      if (res.matched && !res.alreadySwiped && res.conversationId) {
        showMatch({
          matchId: res.matchId ?? "",
          conversationId: res.conversationId,
          peerId: userId,
          peerName: name
        });
      }
      navigation.goBack();
    },
    onError: (err) => toast.show(errorMessage(err), "error")
  });

  if (query.isLoading) return <LoadingState />;
  if (query.isError || !query.data) {
    return (
      <ErrorState message="Ce profil n'est plus disponible." onRetry={() => navigation.goBack()} />
    );
  }
  const p = query.data;
  const photoWidth = Math.min(width, 520);

  return (
    <Screen scroll padded={false}>
      <View>
        <FlatList
          data={p.photos}
          horizontal
          pagingEnabled
          showsHorizontalScrollIndicator={false}
          keyExtractor={(ph) => ph.id}
          onMomentumScrollEnd={(e) =>
            setIndex(Math.round(e.nativeEvent.contentOffset.x / photoWidth))
          }
          style={{ width: photoWidth, alignSelf: "center" }}
          renderItem={({ item }) => (
            <PhotoImage
              photoId={item.id}
              style={{ width: photoWidth, height: photoWidth * 1.25 }}
              accessibilityLabel={`Photo de ${p.firstName}`}
            />
          )}
        />
        {p.photos.length > 1 ? (
          <View style={styles.dots} pointerEvents="none">
            {p.photos.map((ph, i) => (
              <View key={ph.id} style={[styles.dot, { opacity: i === index ? 1 : 0.4 }]} />
            ))}
          </View>
        ) : null}
      </View>
      <View style={styles.body}>
        <Text variant="title">
          {p.firstName}, {p.age}
        </Text>
        <View style={styles.row}>
          <Ionicons name="person-outline" size={16} color={t.textMuted} />
          <Text tone="muted">{GENDER_LABELS[p.gender]}</Text>
        </View>
        {p.city || p.distanceKm !== null ? (
          <View style={styles.row}>
            <Ionicons name="location-outline" size={16} color={t.textMuted} />
            <Text tone="muted">
              {[p.city, p.distanceKm !== null ? `à environ ${p.distanceKm} km` : ""]
                .filter(Boolean)
                .join(" · ")}
            </Text>
          </View>
        ) : null}
        {p.bio ? <Text>{p.bio}</Text> : null}
        {p.interests.length > 0 ? (
          <View style={styles.tags}>
            {p.interests.map((slug) => (
              <Chip key={slug} label={labels.get(slug) ?? slug} />
            ))}
          </View>
        ) : null}
        <SafetyMenu
          userId={userId}
          name={p.firstName}
          onDone={() => {
            qc.removeQueries({ queryKey: queryKeys.feed });
            void qc.invalidateQueries({ queryKey: queryKeys.conversations });
            navigation.popToTop();
          }}
        />
      </View>
      {canSwipe ? (
        <View style={[styles.footer, { backgroundColor: t.bg, borderColor: t.border }]}>
          <Button
            label="Passer"
            variant="secondary"
            style={styles.flex}
            loading={swipe.isPending && swipe.variables === "pass"}
            onPress={() => swipe.mutate("pass")}
          />
          <Button
            label="J'aime"
            style={styles.flex}
            loading={swipe.isPending && swipe.variables === "like"}
            onPress={() => swipe.mutate("like")}
          />
        </View>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  dots: { position: "absolute", bottom: 12, alignSelf: "center", flexDirection: "row", gap: 6 },
  dot: { width: 8, height: 8, borderRadius: 4, backgroundColor: "#fff" },
  body: { padding: 20, gap: 12 },
  row: { flexDirection: "row", alignItems: "center", gap: 8 },
  tags: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  footer: { flexDirection: "row", gap: 12, padding: 16, borderTopWidth: 1 },
  flex: { flex: 1 }
});
