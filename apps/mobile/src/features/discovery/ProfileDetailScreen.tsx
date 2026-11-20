import React, { useState } from "react";
import { ScrollView, StyleSheet, useWindowDimensions, View } from "react-native";
import { Image } from "expo-image";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { mediaUrl } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { CardSummary } from "../../api/types";
import { errorMessage, formatDistance, genderLabels, goalLabels } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { keys } from "../../providers/queryClient";
import { useTheme } from "../../theme";
import { ActionSheet, Badge, Chip, ErrorState, Header, IconButton, LoadingState, Screen, SectionTitle, Text } from "../../ui";
import { useBlock } from "../safety/useBlock";
import { MatchModal } from "./MatchModal";
import { useSwipe } from "./useSwipe";
import { useMyProfile } from "../profile/queries";

type Props = NativeStackScreenProps<AppStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, fromDiscovery } = route.params;
  const { colors, radii, spacing } = useTheme();
  const { width } = useWindowDimensions();
  const queryClient = useQueryClient();
  const me = useMyProfile();
  const [menu, setMenu] = useState(false);
  const [match, setMatch] = useState<{ user: CardSummary; conversationId: string } | null>(null);
  const card = useQuery({ queryKey: keys.profileCard(userId), queryFn: () => profileApi.view(userId) });
  const block = useBlock(() => navigation.goBack());
  const swipe = useSwipe((result) => {
    if (result.matchedUser && result.conversationId) {
      setMatch({ user: result.matchedUser, conversationId: result.conversationId });
    } else {
      navigation.goBack();
    }
  });

  if (card.isLoading) {
    return (
      <Screen>
        <Header back title="" />
        <LoadingState />
      </Screen>
    );
  }
  if (card.isError || !card.data) {
    return (
      <Screen>
        <Header back title="Profil" />
        <ErrorState message={errorMessage(card.error, "Ce profil n’est plus disponible.")} />
      </Screen>
    );
  }
  const p = card.data;
  const photoWidth = Math.min(width, 640) - spacing.lg * 2;
  const facts = [
    p.jobTitle && { icon: "briefcase-outline" as const, text: p.jobTitle },
    p.city && { icon: "location-outline" as const, text: p.city },
    formatDistance(p.distanceKm) && { icon: "navigate-outline" as const, text: formatDistance(p.distanceKm) as string },
    p.relationshipGoal && { icon: "heart-outline" as const, text: goalLabels[p.relationshipGoal] },
    p.heightCm && { icon: "resize-outline" as const, text: `${p.heightCm} cm` },
    { icon: "person-outline" as const, text: genderLabels[p.gender] }
  ].filter(Boolean) as { icon: keyof typeof Ionicons.glyphMap; text: string }[];

  return (
    <Screen>
      <Header
        back
        right={<IconButton icon="ellipsis-horizontal" label="Plus d’options" onPress={() => setMenu(true)} testID="profile-menu" />}
        title={p.firstName}
      />
      <ScrollView contentContainerStyle={{ padding: spacing.lg, gap: spacing.lg, paddingBottom: 140 }}>
        <View style={{ gap: spacing.xs }}>
          {p.isDemo ? <Badge label="Profil de démonstration" tone="accent" /> : null}
          <Text variant="display">
            {p.firstName}, {p.age}
          </Text>
        </View>
        {p.photos.map((photo, i) => (
          <React.Fragment key={photo.id}>
            <Image
              accessibilityLabel={`Photo ${i + 1} de ${p.firstName}`}
              cachePolicy="memory-disk"
              contentFit="cover"
              source={{ uri: mediaUrl(photo.url) }}
              style={{ width: photoWidth, height: photoWidth * (4 / 3), borderRadius: radii.lg, backgroundColor: colors.surfaceAlt }}
              transition={150}
            />
            {i === 0 && p.bio ? (
              <View style={{ gap: spacing.xs }}>
                <SectionTitle>À propos</SectionTitle>
                <Text>{p.bio}</Text>
              </View>
            ) : null}
            {i === 0 ? (
              <View style={styles.facts}>
                {facts.map((f) => (
                  <View key={f.text} style={[styles.fact, { backgroundColor: colors.surface, borderColor: colors.border, borderRadius: radii.pill }]}>
                    <Ionicons color={colors.textMuted} name={f.icon} size={16} />
                    <Text tone="muted" variant="caption">
                      {f.text}
                    </Text>
                  </View>
                ))}
              </View>
            ) : null}
            {i === 0 && p.interests.length ? (
              <View style={{ gap: spacing.xs }}>
                <SectionTitle>Centres d’intérêt</SectionTitle>
                <View style={styles.facts}>
                  {p.interests.map((interest) => (
                    <Chip highlighted={interest.shared} key={interest.id} label={interest.shared ? `♥ ${interest.label}` : interest.label} />
                  ))}
                </View>
              </View>
            ) : null}
          </React.Fragment>
        ))}
        <Text center tone="subtle" variant="caption">
          Un comportement te met mal à l’aise ? Utilise le menu « … » pour bloquer ou signaler.
        </Text>
      </ScrollView>
      {fromDiscovery ? (
        <View style={[styles.floating, { bottom: spacing.xl, gap: spacing.xl }]}>
          <IconButton filled icon="close" label="Passer" onPress={() => swipe.mutate({ userId, action: "pass" })} size={64} />
          <IconButton
            filled
            icon="heart"
            label="J’aime"
            onPress={() => swipe.mutate({ userId, action: "like" })}
            size={64}
            testID="profile-like"
            tone="primary"
          />
        </View>
      ) : null}
      <ActionSheet
        actions={[
          { label: `Bloquer ${p.firstName}`, destructive: true, onPress: () => block.mutate(userId) },
          { label: `Signaler ${p.firstName}`, destructive: true, onPress: () => navigation.navigate("Report", { userId, firstName: p.firstName }) }
        ]}
        message="Bloquer met fin au match et empêche tout contact. La personne n’est pas prévenue."
        onClose={() => setMenu(false)}
        title="Sécurité"
        visible={menu}
      />
      <MatchModal
        match={match}
        myName={me.data?.firstName ?? ""}
        myPhotoUrl={me.data?.photos[0]?.url}
        onClose={() => {
          setMatch(null);
          navigation.goBack();
        }}
        onMessage={(conversationId) => {
          setMatch(null);
          void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
          navigation.replace("Chat", { conversationId });
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  facts: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  fact: { flexDirection: "row", alignItems: "center", gap: 6, paddingHorizontal: 12, paddingVertical: 7, borderWidth: 1 },
  floating: { position: "absolute", left: 0, right: 0, flexDirection: "row", justifyContent: "center" }
});
