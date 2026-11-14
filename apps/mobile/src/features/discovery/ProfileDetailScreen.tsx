import React, { useState } from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { MainStackParamList } from "../../navigation/types";
import { useAuth } from "../../state/AuthProvider";
import { usePublicProfile } from "../../state/queries";
import { radii, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Chip, ErrorState, Icon, IconButton, LoadingState, RemotePhoto, Text } from "../../ui";
import type { IconName } from "../../ui";
import { distanceLabel } from "../../utils/format";
import { lookingForLabels } from "../../utils/labels";
import { errorMessage } from "../../api/client";
import { SafetyMenu } from "../safety/SafetyMenu";
import { SafeAreaView } from "react-native-safe-area-context";
import { deckEvents } from "./deckEvents";

type Props = NativeStackScreenProps<MainStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ navigation, route }: Props) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const { userId, matchId } = route.params;
  const { user: me } = useAuth();
  const isSelf = me?.id === userId;
  const query = usePublicProfile(userId);
  const profile = query.data ?? route.params.profile;
  const [menu, setMenu] = useState(false);
  const fromDeck = !matchId && !!route.params.profile;
  const photoWidth = Math.min(width, 560);

  if (!profile) {
    if (query.isError) return <ErrorState message={errorMessage(query.error)} onRetry={() => navigation.goBack()} />;
    return <LoadingState />;
  }

  const facts: { icon: IconName; text: string }[] = [];
  if (profile.jobTitle) facts.push({ icon: "briefcase-outline", text: profile.jobTitle });
  const dist = distanceLabel(profile.distanceKm);
  if (dist || profile.city) facts.push({ icon: "location-outline", text: [profile.city, dist].filter(Boolean).join(" · ") });
  if (profile.heightCm) facts.push({ icon: "resize-outline", text: `${profile.heightCm} cm` });
  facts.push({ icon: "heart-circle-outline", text: lookingForLabels[profile.lookingFor] });

  const decide = (action: "like" | "pass") => {
    deckEvents.emit({ userId, action });
    navigation.goBack();
  };

  return (
    <SafeAreaView edges={["bottom"]} style={[styles.flex, { backgroundColor: colors.background }]}>
      <ScrollView contentContainerStyle={styles.scroll}>
        <ScrollView decelerationRate="fast" horizontal pagingEnabled showsHorizontalScrollIndicator={false} style={{ width: photoWidth, alignSelf: "center" }}>
          {profile.photos.map((p, i) => (
            <RemotePhoto accessibilityLabel={`Photo ${i + 1} de ${profile.firstName}`} key={p.id} photo={p} style={{ width: photoWidth, height: photoWidth * 1.25 }} />
          ))}
        </ScrollView>
        <View style={styles.body}>
          <Text accessibilityRole="header" variant="display">
            {profile.firstName}, {profile.age}
          </Text>
          <View style={styles.facts}>
            {facts.map((f) => (
              <View key={f.text} style={styles.fact}>
                <Icon color={colors.textMuted} name={f.icon} size={18} />
                <Text tone="muted">{f.text}</Text>
              </View>
            ))}
          </View>
          {profile.bio ? (
            <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
              <Text tone="subtle" variant="overline">
                À propos
              </Text>
              <Text>{profile.bio}</Text>
            </View>
          ) : null}
          {profile.interests.length ? (
            <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
              <Text tone="subtle" variant="overline">
                Centres d’intérêt
              </Text>
              <View style={styles.chips}>
                {profile.interests.map((i) => (
                  <Chip key={i.slug} label={i.label} size="sm" />
                ))}
              </View>
            </View>
          ) : null}
          <Text center tone="subtle" variant="caption">
            Lueur n’affiche jamais la position exacte des membres.
          </Text>
        </View>
      </ScrollView>
      <View style={styles.topBar} pointerEvents="box-none">
        <IconButton icon="chevron-down" label="Fermer" onPress={navigation.goBack} tone="scrim" />
        {isSelf ? null : <IconButton icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setMenu(true)} tone="scrim" />}
      </View>
      {fromDeck ? (
        <View style={styles.actions}>
          <IconButton icon="close" label="Passer" onPress={() => decide("pass")} size={60} tone="surface" />
          <IconButton icon="heart" label="J'aime" onPress={() => decide("like")} size={60} tone="primary" />
        </View>
      ) : null}
      <SafetyMenu
        firstName={profile.firstName}
        matchId={matchId}
        onClose={() => setMenu(false)}
        onRemoved={() => navigation.popToTop()}
        userId={userId}
        visible={menu}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  scroll: { paddingBottom: 120 },
  body: { padding: spacing.lg, gap: spacing.md, width: "100%", maxWidth: 560, alignSelf: "center" },
  facts: { gap: spacing.xs },
  fact: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  card: { padding: spacing.md, borderRadius: radii.lg, borderWidth: StyleSheet.hairlineWidth, gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 6 },
  topBar: { position: "absolute", top: spacing.xxl, left: spacing.md, right: spacing.md, flexDirection: "row", justifyContent: "space-between" },
  actions: { position: "absolute", bottom: spacing.xl, left: 0, right: 0, flexDirection: "row", justifyContent: "center", gap: spacing.xxl }
});
