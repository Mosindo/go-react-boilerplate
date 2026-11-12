import React, { useState } from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { SafetySheet } from "../../components/SafetySheet";
import { useMatches, usePublicProfile } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme";
import { Button, Chip, ErrorState, Header, IconButton, Loading, Photo, Screen, Text } from "../../ui";
import { distanceLabel, genderLabels, goalLabels, nameAge } from "../../utils/format";

export default function ProfileViewScreen({ route, navigation }: NativeStackScreenProps<AppStackParamList, "ProfileView">) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const { userId, fromChat } = route.params;
  const query = usePublicProfile(userId);
  const matches = useMatches();
  const [sheet, setSheet] = useState(false);
  const match = matches.data?.find((m) => m.user.userId === userId);

  if (query.isLoading) return <Loading />;
  if (query.error || !query.data) {
    return (
      <Screen>
        <Header title="" onBack={navigation.goBack} />
        <ErrorState message={errorMessage(query.error, "Ce profil n'est plus disponible.")} onRetry={() => void query.refetch()} />
      </Screen>
    );
  }
  const p = query.data;
  const photoWidth = Math.min(width, 560) - spacing.lg * 2;

  return (
    <Screen padded={false}>
      <Header
        title={p.firstName}
        onBack={navigation.goBack}
        right={<IconButton icon="ellipsis-horizontal" label="Options de sécurité" onPress={() => setSheet(true)} />}
      />
      <ScrollView contentContainerStyle={styles.content}>
        <Photo
          url={p.photos[0]?.url}
          style={[styles.photo, { width: photoWidth, height: photoWidth * 1.25 }]}
          accessibilityLabel={`Photo principale de ${p.firstName}`}
        />
        <View style={styles.block}>
          <Text variant="title" serif>
            {nameAge(p.firstName, p.age)}
          </Text>
          <Text tone="muted">{[genderLabels[p.gender], p.occupation, p.city, distanceLabel(p.distanceKm)].filter(Boolean).join(" · ")}</Text>
        </View>
        <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
          <Text variant="overline" tone="subtle">
            Recherche
          </Text>
          <Text variant="bodyStrong">{goalLabels[p.relationshipGoal]}</Text>
        </View>
        {p.bio ? (
          <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
            <Text variant="overline" tone="subtle">
              À propos
            </Text>
            <Text>{p.bio}</Text>
          </View>
        ) : null}
        {p.photos.slice(1).map((photo) => (
          <Photo key={photo.id} url={photo.url} style={[styles.photo, { width: photoWidth, height: photoWidth * 1.25 }]} />
        ))}
        {p.interests.length ? (
          <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
            <Text variant="overline" tone="subtle">
              Centres d’intérêt
            </Text>
            <View style={styles.chips}>
              {p.interests.map((i) => (
                <Chip key={i.id} label={i.label} />
              ))}
            </View>
          </View>
        ) : null}
        {match && !fromChat ? (
          <Button label={`Écrire à ${p.firstName}`} icon="chatbubble-ellipses" onPress={() => navigation.replace("Chat", { userId, firstName: p.firstName })} />
        ) : null}
        <Button label={`Bloquer ou signaler ${p.firstName}`} variant="ghost" onPress={() => setSheet(true)} />
      </ScrollView>
      <SafetySheet
        visible={sheet}
        onClose={() => setSheet(false)}
        userId={userId}
        firstName={p.firstName}
        matchId={match?.id}
        onDone={() => navigation.navigate("Tabs")}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.lg, gap: spacing.md, alignItems: "stretch", paddingBottom: spacing.xxxl },
  photo: { borderRadius: radii.xl, alignSelf: "center" },
  block: { gap: 4 },
  card: { borderRadius: radii.lg, borderWidth: 1, padding: spacing.md, gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
