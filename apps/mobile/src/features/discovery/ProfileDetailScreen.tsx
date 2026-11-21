import React, { useRef, useState } from "react";
import { ScrollView, View, useWindowDimensions, type NativeScrollEvent, type NativeSyntheticEvent } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { SwipeAction } from "../../api/types";
import type { RootStackParamList } from "../../navigation/types";
import { makeStyles } from "../../theme";
import { Button, Chip, ErrorState, Icon, IconButton, LoadingState, RemoteImage, Section, Text } from "../../ui";
import { formatDistance, genderLabels, goalLabels } from "../../utils/format";
import { SafetyMenu, type SafetyMenuHandle } from "../safety/SafetyMenu";

type Props = NativeStackScreenProps<RootStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ navigation, route }: Props) {
  const styles = useStyles();
  const { userId, mode } = route.params;
  const { width } = useWindowDimensions();
  const pageWidth = Math.min(width, 640);
  const [page, setPage] = useState(0);
  const safetyRef = useRef<SafetyMenuHandle>(null);
  const query = useQuery({ queryKey: queryKeys.profile(userId), queryFn: () => profileApi.view(userId) });

  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError || !query.data) {
    return (
      <SafeAreaView style={styles.root}>
        <ErrorState message={errorMessage(query.error, "Ce profil n'est plus disponible.")} onRetry={() => navigation.goBack()} />
      </SafeAreaView>
    );
  }
  const card = query.data;
  const decide = (action: SwipeAction) =>
    navigation.navigate("Tabs", { screen: "Discover", params: { decision: { userId, action, nonce: Date.now() } } });

  const onScroll = (e: NativeSyntheticEvent<NativeScrollEvent>) => setPage(Math.round(e.nativeEvent.contentOffset.x / pageWidth));
  const facts = [
    card.jobTitle ? { icon: "briefcase-outline" as const, text: card.jobTitle } : null,
    card.city ? { icon: "home-outline" as const, text: card.city } : null,
    formatDistance(card.distanceKm) ? { icon: "location-outline" as const, text: formatDistance(card.distanceKm) as string } : null,
    card.heightCm ? { icon: "resize-outline" as const, text: `${card.heightCm} cm` } : null,
    { icon: "person-outline" as const, text: genderLabels[card.gender] },
    card.relationshipGoal ? { icon: "sparkles-outline" as const, text: goalLabels[card.relationshipGoal] } : null
  ].filter((f): f is { icon: "briefcase-outline"; text: string } => f !== null);

  return (
    <SafeAreaView style={styles.root} edges={["top"]}>
      <View style={styles.topBar}>
        <IconButton icon="chevron-down" accessibilityLabel="Fermer" onPress={() => navigation.goBack()} />
        <IconButton
          testID="profile-menu"
          icon="ellipsis-horizontal"
          accessibilityLabel="Options de sécurité"
          onPress={() => safetyRef.current?.open()}
        />
      </View>
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={[styles.carousel, { width: pageWidth }]}>
          <ScrollView
            horizontal
            pagingEnabled
            showsHorizontalScrollIndicator={false}
            onMomentumScrollEnd={onScroll}
            onScroll={onScroll}
            scrollEventThrottle={64}
          >
            {card.photos.map((photo, i) => (
              <RemoteImage
                key={photo.id}
                uri={photo.url}
                style={{ width: pageWidth, aspectRatio: 4 / 5 }}
                accessibilityLabel={`Photo ${i + 1} de ${card.firstName}`}
              />
            ))}
          </ScrollView>
          {card.photos.length > 1 ? (
            <View style={styles.dots}>
              {card.photos.map((p, i) => (
                <View key={p.id} style={[styles.dot, i === page ? styles.dotActive : null]} />
              ))}
            </View>
          ) : null}
        </View>
        <View style={styles.body}>
          <View style={styles.nameRow}>
            <Text variant="display" accessibilityRole="header">
              {card.firstName}, {card.age}
            </Text>
            {card.isDemo ? (
              <View style={styles.demo}>
                <Text variant="micro" tone="muted">
                  DÉMO
                </Text>
              </View>
            ) : null}
          </View>
          <View style={styles.facts}>
            {facts.map((f) => (
              <View key={f.text} style={styles.fact}>
                <Icon name={f.icon} size={18} color="textMuted" />
                <Text tone="muted">{f.text}</Text>
              </View>
            ))}
          </View>
          {card.bio ? (
            <Section title="À propos">
              <Text>{card.bio}</Text>
            </Section>
          ) : null}
          {card.interests.length > 0 ? (
            <Section title="Centres d'intérêt">
              <View style={styles.chips}>
                {card.interests.map((i) => (
                  <Chip key={i.id} label={i.label} />
                ))}
              </View>
            </Section>
          ) : null}
        </View>
      </ScrollView>
      {mode === "discovery" ? (
        <View style={styles.actions}>
          <IconButton
            testID="profile-pass"
            icon="close"
            accessibilityLabel={`Passer ${card.firstName}`}
            size={60}
            tone="pass"
            style={styles.passButton}
            onPress={() => decide("pass")}
          />
          <IconButton
            testID="profile-like"
            icon="heart"
            accessibilityLabel={`J'aime ${card.firstName}`}
            size={68}
            filled
            tone="like"
            onPress={() => decide("like")}
          />
        </View>
      ) : mode === "match" ? (
        <View style={styles.footer}>
          <Button
            label={`Écrire à ${card.firstName}`}
            icon="chatbubble-ellipses-outline"
            size="lg"
            onPress={() => navigation.navigate("Conversation", { userId, firstName: card.firstName, photoUrl: card.photos[0]?.thumbUrl })}
          />
        </View>
      ) : null}
      <SafetyMenu
        ref={safetyRef}
        userId={userId}
        firstName={card.firstName}
        matched={mode === "match"}
        onRemoved={() => navigation.navigate("Tabs", { screen: mode === "discovery" ? "Discover" : "Matches" })}
      />
    </SafeAreaView>
  );
}

const useStyles = makeStyles((t) => ({
  root: { flex: 1, backgroundColor: t.colors.background },
  topBar: { flexDirection: "row", justifyContent: "space-between", paddingHorizontal: t.spacing.sm },
  scroll: { paddingBottom: 120, alignItems: "center" },
  carousel: { borderRadius: t.radii.lg, overflow: "hidden" },
  dots: { position: "absolute", bottom: t.spacing.md, alignSelf: "center", flexDirection: "row", gap: 6 },
  dot: { width: 7, height: 7, borderRadius: 4, backgroundColor: "rgba(255,255,255,0.5)" },
  dotActive: { backgroundColor: "#FFFFFF" },
  body: { width: "100%", maxWidth: 640, padding: t.spacing.xl, gap: t.spacing.xl },
  nameRow: { flexDirection: "row", alignItems: "center", gap: t.spacing.md, flexWrap: "wrap" },
  demo: { borderWidth: 1, borderColor: t.colors.border, borderRadius: t.radii.pill, paddingHorizontal: t.spacing.sm, paddingVertical: 2 },
  facts: { gap: t.spacing.sm },
  fact: { flexDirection: "row", alignItems: "center", gap: t.spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: t.spacing.sm },
  actions: {
    position: "absolute",
    bottom: t.spacing.xl,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: t.spacing.xxl
  },
  passButton: { backgroundColor: t.colors.surface, borderWidth: 1, borderColor: t.colors.border },
  footer: {
    position: "absolute",
    bottom: 0,
    left: 0,
    right: 0,
    padding: t.spacing.xl,
    backgroundColor: t.colors.background,
    alignItems: "center"
  }
}));
