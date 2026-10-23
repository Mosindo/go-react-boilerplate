import React, { useEffect, useRef, useState } from "react";
import { FlatList, View, useWindowDimensions } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useSwipe } from "../../features/useSwipe";
import { useModeration } from "../../features/useModeration";
import { t } from "../../i18n/fr";
import { formatDistance, genderLabel } from "../../lib/format";
import { useCard } from "../../queries";
import { createStyles, useTheme } from "../../theme/ThemeProvider";
import { Button, ErrorState, LoadingState, Photo, Screen, Tag, TagWrap, Text } from "../../ui";
import type { MainStackParams } from "../navTypes";
import { RoundButton } from "./DiscoverScreen";

type Props = NativeStackScreenProps<MainStackParams, "ProfileDetail">;

const useStyles = createStyles(({ colors, spacing, radius }) => ({
  dots: { flexDirection: "row", justifyContent: "center", gap: 6, paddingVertical: spacing.sm },
  dot: { width: 8, height: 8, borderRadius: 4, backgroundColor: colors.border },
  dotOn: { backgroundColor: colors.primary, width: 20 },
  body: { gap: spacing.md },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xl,
    paddingVertical: spacing.md
  },
  carousel: { borderRadius: radius.lg, overflow: "hidden" }
}));

export default function ProfileDetailScreen({ navigation, route }: Props) {
  const styles = useStyles();
  const { colors } = useTheme();
  const { userId, card: initial, canSwipe } = route.params;
  const query = useCard(userId, initial);
  const { decide } = useSwipe();
  const moderation = useModeration(() => navigation.goBack());
  const { width } = useWindowDimensions();
  const [page, setPage] = useState(0);
  const [boxWidth, setBoxWidth] = useState(Math.min(width, 560) - 32);
  const listRef = useRef<FlatList>(null);
  const card = query.data;

  useEffect(() => {
    if (!card) return;
    navigation.setOptions({
      title: card.firstName,
      headerRight: () => (
        <Button
          accessibilityLabel={t.common.more}
          compact
          label="⋯"
          onPress={() => moderation.openMenu({ userId: card.userId, firstName: card.firstName })}
          testID="profile-menu-button"
          variant="ghost"
        />
      )
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [card, navigation, colors]);

  if (query.isPending) return <LoadingState />;
  if (query.isError || !card) {
    return <ErrorState onRetry={() => query.refetch()} title={t.profileDetail.loadError} />;
  }

  const photoHeight = Math.min(boxWidth * 1.25, 520);
  const distance = formatDistance(card.distanceKm);

  const swipeAndLeave = (action: "like" | "pass") => {
    decide(card, action);
    navigation.goBack();
  };

  return (
    <Screen scroll testID="profile-detail-screen">
      <View onLayout={(e) => setBoxWidth(e.nativeEvent.layout.width)} style={styles.carousel}>
        <FlatList
          data={card.photos}
          getItemLayout={(_d, index) => ({ length: boxWidth, offset: boxWidth * index, index })}
          horizontal
          keyExtractor={(photo) => photo.id}
          onMomentumScrollEnd={(e) => setPage(Math.round(e.nativeEvent.contentOffset.x / boxWidth))}
          pagingEnabled
          ref={listRef}
          renderItem={({ item, index }) => (
            <Photo
              label={`${t.a11y.photoOf(card.firstName)} - ${t.profileDetail.photoOf(index + 1, card.photos.length)}`}
              name={card.firstName}
              path={item.url}
              style={{ width: boxWidth, height: photoHeight }}
            />
          )}
          showsHorizontalScrollIndicator={false}
          testID="profile-carousel"
        />
        {card.photos.length === 0 ? (
          <Photo
            name={card.firstName}
            path={null}
            style={{ width: boxWidth, height: photoHeight }}
          />
        ) : null}
      </View>
      {card.photos.length > 1 ? (
        <View
          accessibilityLabel={t.profileDetail.photoOf(page + 1, card.photos.length)}
          style={styles.dots}
        >
          {card.photos.map((photo, index) => (
            <View key={photo.id} style={[styles.dot, index === page ? styles.dotOn : null]} />
          ))}
        </View>
      ) : null}

      <View style={styles.body}>
        <Text variant="title">
          {card.firstName}, {card.age}
        </Text>
        <Text tone="muted">
          {genderLabel(card.gender)} · {card.city}
          {distance ? ` · ${distance}` : ""}
        </Text>
        <Text variant="heading">{t.profileDetail.about}</Text>
        <Text tone={card.bio ? "default" : "muted"}>{card.bio || t.profileDetail.noBio}</Text>
        {card.interests.length > 0 ? (
          <>
            <Text variant="heading">{t.profileDetail.interests}</Text>
            <TagWrap>
              {card.interests.map((interest) => (
                <Tag key={interest.slug} label={interest.label} shared={interest.shared} />
              ))}
            </TagWrap>
          </>
        ) : null}
      </View>

      {canSwipe ? (
        <View style={styles.actions}>
          <RoundButton
            glyph="✕"
            label={t.discover.passLabel(card.firstName)}
            onPress={() => swipeAndLeave("pass")}
            testID="detail-pass-button"
            tone="pass"
          />
          <RoundButton
            glyph="♥"
            label={t.discover.likeLabel(card.firstName)}
            onPress={() => swipeAndLeave("like")}
            testID="detail-like-button"
            tone="like"
          />
        </View>
      ) : null}
    </Screen>
  );
}
