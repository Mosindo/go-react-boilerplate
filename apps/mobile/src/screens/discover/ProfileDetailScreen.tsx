import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { Ionicons } from "@expo/vector-icons";
import { discoveryApi } from "../../api/social";
import { PhotoCarousel } from "../../components/PhotoCarousel";
import { SafetySheet } from "../../components/SafetySheet";
import { keys, useProfileCard } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { colors, radii, spacing } from "../../theme/tokens";
import { Button, Chip, ErrorState, IconButton, LoadingState, Screen, Text, showToast } from "../../ui";
import { errorMessage } from "../../utils/errors";
import { distanceLabel, genderLabels, lookingForLabels } from "../../utils/format";

export default function ProfileDetailScreen({ navigation, route }: AppScreenProps<"ProfileDetail">) {
  const { userId, card: initial, fromDeck, matchId } = route.params;
  const qc = useQueryClient();
  const query = useProfileCard(userId);
  const card = query.data ?? initial;
  const [safetyOpen, setSafetyOpen] = useState(false);
  const [busy, setBusy] = useState(false);

  if (!card) {
    return <Screen>{query.isLoading ? <LoadingState /> : <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />}</Screen>;
  }

  const decide = async (action: "like" | "pass") => {
    setBusy(true);
    try {
      const res = await discoveryApi.swipe(card.userId, action);
      await qc.invalidateQueries({ queryKey: keys.deck });
      if (res.matched && res.match) {
        showToast(`C'est réciproque avec ${card.firstName} !`, "success");
        void qc.invalidateQueries({ queryKey: keys.matches });
        navigation.replace("Conversation", { matchId: res.match.id });
        return;
      }
      navigation.goBack();
    } catch (err) {
      showToast(errorMessage(err), "error");
    } finally {
      setBusy(false);
    }
  };

  const distance = distanceLabel(card.distanceKm);
  return (
    <Screen scroll padded={false} edges={["top", "bottom"]} testID="profile-detail">
      <View style={styles.photos}>
        <PhotoCarousel photos={card.photos} name={card.firstName} rounded={false} />
        <View style={styles.topBar}>
          <IconButton icon="chevron-down" label="Fermer" onPress={navigation.goBack} background={colors.scrim} />
          <IconButton
            icon="ellipsis-horizontal"
            label="Options de sécurité"
            onPress={() => setSafetyOpen(true)}
            background={colors.scrim}
            testID="profile-safety"
          />
        </View>
      </View>
      <View style={styles.body}>
        <Text variant="display">
          {card.firstName},{" "}
          <Text variant="title" style={{ fontWeight: "400" }}>
            {card.age}
          </Text>
        </Text>
        <View style={styles.facts}>
          {card.jobTitle ? <Fact icon="briefcase-outline" text={card.jobTitle} /> : null}
          {card.city ? <Fact icon="home-outline" text={card.city} /> : null}
          {distance ? <Fact icon="location-outline" text={distance} /> : null}
          <Fact icon="person-outline" text={genderLabels[card.gender]} />
          {card.heightCm ? <Fact icon="resize-outline" text={`${card.heightCm} cm`} /> : null}
          <Fact icon="sparkles-outline" text={lookingForLabels[card.lookingFor]} />
        </View>
        {card.bio ? (
          <View style={styles.block}>
            <Text variant="label" tone="muted">
              À PROPOS
            </Text>
            <Text>{card.bio}</Text>
          </View>
        ) : null}
        {card.interests.length ? (
          <View style={styles.block}>
            <Text variant="label" tone="muted">
              CENTRES D&apos;INTÉRÊT
            </Text>
            <View style={styles.chips}>
              {card.interests.map((i) => (
                <Chip key={i.id} label={i.label} highlighted={i.shared} icon={i.shared ? "heart" : undefined} />
              ))}
            </View>
          </View>
        ) : null}
        {fromDeck ? (
          <View style={styles.decide}>
            <Button label="Passer" variant="secondary" icon="close" onPress={() => decide("pass")} disabled={busy} style={styles.grow} />
            <Button label="J'aime" icon="heart" onPress={() => decide("like")} loading={busy} style={styles.grow} testID="detail-like" />
          </View>
        ) : null}
      </View>
      <SafetySheet
        visible={safetyOpen}
        userId={card.userId}
        firstName={card.firstName}
        matchId={matchId}
        onClose={() => setSafetyOpen(false)}
        onDone={() => {
          void qc.invalidateQueries({ queryKey: keys.deck });
          navigation.navigate("Tabs", { screen: fromDeck ? "Discover" : "Matches" });
        }}
      />
    </Screen>
  );
}

function Fact({ icon, text }: { icon: keyof typeof Ionicons.glyphMap; text: string }) {
  return (
    <View style={styles.fact}>
      <Ionicons name={icon} size={16} color={colors.accent} />
      <Text variant="caption">{text}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  photos: { height: 540, backgroundColor: colors.surface },
  topBar: { position: "absolute", top: spacing.md, left: spacing.md, right: spacing.md, flexDirection: "row", justifyContent: "space-between", zIndex: 5 },
  body: { padding: spacing.xl, gap: spacing.xl },
  facts: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  fact: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radii.pill,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  block: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  decide: { flexDirection: "row", gap: spacing.md },
  grow: { flex: 1 }
});
