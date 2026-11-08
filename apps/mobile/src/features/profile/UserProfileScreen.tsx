import React, { useState } from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootParamList } from "../../navigation/types";
import { useUserProfile } from "../../hooks/queries";
import { formatDistance } from "../../lib/format";
import { radii, spacing, useTheme } from "../../theme/theme";
import { IconButton } from "../../ui/Button";
import { Chip } from "../../ui/Chip";
import { ErrorState, LoadingView } from "../../ui/Feedback";
import { Photo } from "../../ui/Photo";
import { Screen } from "../../ui/Screen";
import { Text } from "../../ui/Text";
import { useSafetyMenu } from "../safety/useSafetyMenu";
import { useSwipe } from "../discover/useSwipe";

type Props = NativeStackScreenProps<RootParamList, "UserProfile">;

export function UserProfileScreen({ navigation, route }: Props) {
  const { userId, card: initial, canSwipe } = route.params;
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const query = useUserProfile(userId);
  const swipe = useSwipe();
  const [page, setPage] = useState(0);
  const card = query.data ?? initial;
  const safety = useSafetyMenu({
    userId,
    firstName: card?.firstName ?? "cette personne",
    onBlocked: () => navigation.popToTop()
  });

  if (!card) {
    return (
      <Screen>
        {query.isError ? (
          <ErrorState message={query.error.message} onRetry={() => void query.refetch()} />
        ) : (
          <LoadingView />
        )}
      </Screen>
    );
  }
  const distance = formatDistance(card.distanceKm);
  const photoHeight = Math.min(width * 1.25, 560);

  const decide = (action: "like" | "pass") => {
    swipe.mutate({ card, action });
    navigation.goBack();
  };

  return (
    <Screen padded={false} edges={["bottom"]}>
      <ScrollView showsVerticalScrollIndicator={false}>
        <ScrollView
          horizontal
          pagingEnabled
          showsHorizontalScrollIndicator={false}
          onMomentumScrollEnd={(e) => setPage(Math.round(e.nativeEvent.contentOffset.x / width))}
          style={{ height: photoHeight }}
        >
          {card.photos.map((p, i) => (
            <Photo
              key={p.id}
              path={p.url}
              style={{ width, height: photoHeight }}
              accessibilityLabel={`Photo ${i + 1} de ${card.firstName}`}
            />
          ))}
        </ScrollView>
        {card.photos.length > 1 ? (
          <View style={styles.dots} pointerEvents="none">
            {card.photos.map((p, i) => (
              <View
                key={p.id}
                style={[styles.dot, { backgroundColor: i === page ? colors.primary : colors.border }]}
              />
            ))}
          </View>
        ) : null}

        <View style={styles.body}>
          <View style={styles.titleRow}>
            <Text variant="display" style={styles.flex}>
              {card.firstName}, {card.age}
            </Text>
            <IconButton
              icon="ellipsis-horizontal"
              label="Plus d'options"
              onPress={safety.open}
              testID="profile-menu"
            />
          </View>
          <View style={styles.meta}>
            <Ionicons name="location-outline" size={16} color={colors.textMuted} />
            <Text tone="textMuted">{[card.city, distance].filter(Boolean).join(" · ")}</Text>
          </View>

          {card.bio ? (
            <>
              <Text variant="label" tone="textSubtle" style={styles.section}>
                À PROPOS
              </Text>
              <Text>{card.bio}</Text>
            </>
          ) : null}

          {card.interests.length > 0 ? (
            <>
              <Text variant="label" tone="textSubtle" style={styles.section}>
                CENTRES D&apos;INTÉRÊT
              </Text>
              <View style={styles.chips}>
                {card.interests.map((i) => (
                  <Chip key={i.slug} label={i.label} />
                ))}
              </View>
            </>
          ) : null}
        </View>
      </ScrollView>

      {canSwipe ? (
        <View style={[styles.actions, { backgroundColor: colors.background, borderTopColor: colors.border }]}>
          <IconButton
            icon="close"
            label="Passer"
            size={60}
            color={colors.pass}
            background={colors.surface}
            onPress={() => decide("pass")}
            testID="profile-pass"
          />
          <IconButton
            icon="heart"
            label="J'aime"
            size={68}
            color={colors.onPrimary}
            background={colors.primary}
            onPress={() => decide("like")}
            testID="profile-like"
          />
        </View>
      ) : null}
      {safety.element}
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  dots: { flexDirection: "row", justifyContent: "center", gap: 6, marginTop: spacing.xs },
  dot: { width: 8, height: 8, borderRadius: 4 },
  body: { padding: spacing.md },
  titleRow: { flexDirection: "row", alignItems: "center" },
  meta: { flexDirection: "row", alignItems: "center", gap: 4, marginTop: 2 },
  section: { marginTop: spacing.lg, marginBottom: spacing.xs, letterSpacing: 1 },
  chips: { flexDirection: "row", flexWrap: "wrap" },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xl,
    paddingVertical: spacing.sm,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderRadius: radii.sm
  }
});
