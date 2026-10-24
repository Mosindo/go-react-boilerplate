import React, { useCallback, useState } from "react";
import { FlatList, Pressable, StyleSheet, useWindowDimensions, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import type { MainStackParamList } from "../../navigation/types";
import { queryKeys } from "../../realtime/cache";
import { radius, spacing, useTheme } from "../../theme/theme";
import { Chip, ErrorState, Icon, LoadingState, Photo, Screen, Text, useDialog } from "../../ui";
import { useSafetyActions } from "../safety/useSafetyActions";
import { useSwipe } from "./useSwipe";

type Props = NativeStackScreenProps<MainStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ route, navigation }: Props) {
  const { userId, canSwipe } = route.params;
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const dialog = useDialog();
  const swipe = useSwipe();
  const [page, setPage] = useState(0);
  const {
    data: card,
    error,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: queryKeys.card(userId),
    queryFn: () => profileApi.card(userId),
  });
  const goBack = useCallback(() => navigation.goBack(), [navigation]);
  const safety = useSafetyActions(userId, card?.firstName ?? "cette personne", goBack);

  if (isLoading) return <LoadingState />;
  if (error || !card) return <ErrorState error={error} onRetry={() => void refetch()} />;

  const decide = async (action: "like" | "pass") => {
    if (await swipe(userId, action)) {
      if (!navigation.canGoBack()) return;
      // A match navigates to MatchFound by itself; otherwise return to the deck.
      if (navigation.getState().routes.at(-1)?.name === "ProfileDetail") navigation.goBack();
    }
  };

  const openMenu = async () => {
    const choice = await dialog.choose({
      title: card.firstName,
      actions: [
        { label: "Signaler", value: "report", variant: "secondary" },
        { label: "Bloquer", value: "block", variant: "danger" },
        { label: "Annuler", value: "cancel", variant: "ghost" },
      ],
    });
    if (choice === "report") await safety.report();
    if (choice === "block") await safety.block();
  };

  const place = [card.city, card.distanceKm ? `à ${card.distanceKm} km` : null]
    .filter(Boolean)
    .join(" · ");

  return (
    <Screen testID="profile-detail" scroll padded={false} edges={["bottom"]}>
      <View>
        <FlatList
          data={card.photos}
          keyExtractor={(p) => p.id}
          horizontal
          pagingEnabled
          showsHorizontalScrollIndicator={false}
          onMomentumScrollEnd={(e) => setPage(Math.round(e.nativeEvent.contentOffset.x / width))}
          renderItem={({ item }) => (
            <Photo
              photoId={item.id}
              style={{ width, height: width * 1.25 }}
              accessibilityLabel={`Photo de ${card.firstName}`}
            />
          )}
        />
        {card.photos.length > 1 ? (
          <View style={styles.pager} pointerEvents="none">
            {card.photos.map((p, i) => (
              <View key={p.id} style={[styles.pageDot, { opacity: i === page ? 1 : 0.4 }]} />
            ))}
          </View>
        ) : null}
        <Pressable
          testID="detail-back"
          accessibilityRole="button"
          accessibilityLabel="Retour"
          onPress={() => navigation.goBack()}
          style={[styles.floating, { left: spacing.lg }]}
        >
          <Icon name="chevron-back" size={24} color="#FFFFFF" />
        </Pressable>
        <Pressable
          testID="detail-menu"
          accessibilityRole="button"
          accessibilityLabel="Plus d'options"
          onPress={openMenu}
          style={[styles.floating, { right: spacing.lg }]}
        >
          <Icon name="ellipsis-horizontal" size={24} color="#FFFFFF" />
        </Pressable>
      </View>

      <View style={styles.content}>
        <Text variant="title" testID="detail-name">
          {card.firstName}, {card.age}
        </Text>
        {place ? <Text muted>{place}</Text> : null}
        {card.bio ? <Text>{card.bio}</Text> : null}
        {card.interests.length > 0 ? (
          <View style={styles.chips}>
            {card.interests.map((i) => (
              <Chip key={i.slug} label={i.label} />
            ))}
          </View>
        ) : null}
      </View>

      {canSwipe ? (
        <View style={styles.actions}>
          <Pressable
            testID="detail-pass"
            accessibilityRole="button"
            accessibilityLabel="Passer"
            onPress={() => decide("pass")}
            style={[styles.round, { borderColor: colors.danger, backgroundColor: colors.surface }]}
          >
            <Icon name="close" size={32} color={colors.danger} />
          </Pressable>
          <Pressable
            testID="detail-like"
            accessibilityRole="button"
            accessibilityLabel="J'aime"
            onPress={() => decide("like")}
            style={[styles.round, { borderColor: colors.success, backgroundColor: colors.surface }]}
          >
            <Icon name="heart" size={32} color={colors.success} />
          </Pressable>
        </View>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  pager: {
    position: "absolute",
    bottom: spacing.md,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: 6,
  },
  pageDot: { width: 8, height: 8, borderRadius: 4, backgroundColor: "#FFFFFF" },
  floating: {
    position: "absolute",
    top: spacing.lg,
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: "rgba(0,0,0,0.4)",
    alignItems: "center",
    justifyContent: "center",
  },
  content: { padding: spacing.lg, gap: spacing.md },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xxl,
    paddingBottom: spacing.xl,
  },
  round: {
    width: 68,
    height: 68,
    borderRadius: radius.pill,
    borderWidth: 2,
    alignItems: "center",
    justifyContent: "center",
  },
});
