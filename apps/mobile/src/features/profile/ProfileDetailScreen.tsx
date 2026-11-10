import React, { useState } from "react";
import { Alert, Platform, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { useQueryClient } from "@tanstack/react-query";
import { Ionicons } from "@expo/vector-icons";
import { errorMessage, mediaSource } from "../../api/client";
import { matchesApi, safetyApi } from "../../api/endpoints";
import { keys, useProfileCard } from "../../hooks/queries";
import type { MainScreenProps } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Chip, ErrorState, Header, LoadingState, Screen, Text, showToast } from "../../ui";
import { formatDistance, genderLabels, goalLabels } from "../../utils/format";
import { ReportSheet } from "../safety/ReportSheet";

/** Cross-platform confirmation (Alert is not available on web). */
export function confirm(title: string, message: string, confirmLabel: string): Promise<boolean> {
  if (Platform.OS === "web") return Promise.resolve(globalThis.confirm?.(`${title}\n\n${message}`) ?? true);
  return new Promise((resolve) =>
    Alert.alert(title, message, [
      { text: "Annuler", style: "cancel", onPress: () => resolve(false) },
      { text: confirmLabel, style: "destructive", onPress: () => resolve(true) }
    ])
  );
}

export function ProfileDetailScreen({ route, navigation }: MainScreenProps<"ProfileDetail">) {
  const { userId, matchId } = route.params;
  const theme = useTheme();
  const qc = useQueryClient();
  const { data: card, isLoading, error, refetch } = useProfileCard(userId);
  const [reporting, setReporting] = useState(false);

  const leave = () => {
    void qc.invalidateQueries({ queryKey: keys.matches });
    navigation.navigate("Tabs");
  };

  const block = async () => {
    if (!card) return;
    if (!(await confirm(`Bloquer ${card.firstName} ?`, "Vous ne vous verrez plus et votre conversation sera supprimée.", "Bloquer"))) return;
    try {
      await safetyApi.block(userId);
      showToast(`${card.firstName} est bloqué·e.`, "success");
      leave();
    } catch (e) {
      showToast(errorMessage(e), "error");
    }
  };

  const unmatch = async () => {
    if (!card || !matchId) return;
    if (!(await confirm(`Ne plus matcher avec ${card.firstName} ?`, "La conversation sera supprimée pour vous deux.", "Confirmer"))) return;
    try {
      await matchesApi.unmatch(matchId);
      leave();
    } catch (e) {
      showToast(errorMessage(e), "error");
    }
  };

  if (isLoading)
    return (
      <Screen>
        <Header back />
        <LoadingState />
      </Screen>
    );
  if (error || !card) {
    return (
      <Screen>
        <Header back />
        <ErrorState message={errorMessage(error, "Ce profil n'est plus disponible.")} onRetry={() => refetch()} />
      </Screen>
    );
  }

  const facts = [
    card.jobTitle ? { icon: "briefcase-outline" as const, text: card.jobTitle } : null,
    card.city ? { icon: "home-outline" as const, text: card.city } : null,
    card.distanceKm !== null ? { icon: "location-outline" as const, text: formatDistance(card.distanceKm) ?? "" } : null,
    card.heightCm ? { icon: "resize-outline" as const, text: `${card.heightCm} cm` } : null,
    card.relationshipGoal ? { icon: "heart-outline" as const, text: goalLabels[card.relationshipGoal] } : null,
    { icon: "person-outline" as const, text: genderLabels[card.gender] }
  ].filter((f): f is { icon: "person-outline"; text: string } => !!f);

  return (
    <Screen scroll padded={false}>
      <View style={styles.headerPad}>
        <Header back title={card.firstName} />
      </View>
      <View style={styles.content}>
        {card.photos[0] ? (
          <Image
            accessibilityLabel={`Photo principale de ${card.firstName}`}
            contentFit="cover"
            source={mediaSource(card.photos[0])}
            style={[styles.photo, { borderRadius: theme.radii.xl }]}
          />
        ) : null}
        <View style={styles.section}>
          <Text variant="display">
            {card.firstName}, {card.age}
          </Text>
          <View style={styles.facts}>
            {facts.map((f) => (
              <View key={f.text} style={styles.fact}>
                <Ionicons color={theme.colors.textMuted} name={f.icon} size={16} />
                <Text tone="muted" variant="small">
                  {f.text}
                </Text>
              </View>
            ))}
          </View>
        </View>
        {card.bio ? (
          <View style={[styles.block, { backgroundColor: theme.colors.surface, borderColor: theme.colors.border }]}>
            <Text tone="muted" variant="label">
              À PROPOS
            </Text>
            <Text>{card.bio}</Text>
          </View>
        ) : null}
        {card.interests.length ? (
          <View style={styles.section}>
            <Text tone="muted" variant="label">
              CENTRES D’INTÉRÊT
            </Text>
            <View style={styles.chips}>
              {card.interests.map((i) => (
                <Chip key={i.id} label={i.label} />
              ))}
            </View>
          </View>
        ) : null}
        {card.photos.slice(1).map((p) => (
          <Image
            accessibilityLabel={`Photo de ${card.firstName}`}
            contentFit="cover"
            key={p.id}
            source={mediaSource(p)}
            style={[styles.photo, { borderRadius: theme.radii.xl }]}
          />
        ))}
        <View style={styles.actions}>
          {matchId ? <Button label="Ne plus matcher" onPress={unmatch} variant="secondary" /> : null}
          <Button icon="ban-outline" label={`Bloquer ${card.firstName}`} onPress={block} testID="profile-block" variant="secondary" />
          <Button icon="flag-outline" label="Signaler" onPress={() => setReporting(true)} testID="profile-report" variant="danger" />
        </View>
      </View>
      <ReportSheet
        name={card.firstName}
        onClose={() => setReporting(false)}
        onDone={() => {
          setReporting(false);
          leave();
        }}
        userId={userId}
        visible={reporting}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  headerPad: { paddingHorizontal: 12 },
  content: { gap: 20, paddingHorizontal: 20, width: "100%", maxWidth: 560, alignSelf: "center" },
  photo: { width: "100%", aspectRatio: 4 / 5 },
  section: { gap: 10 },
  facts: { flexDirection: "row", flexWrap: "wrap", gap: 14 },
  fact: { flexDirection: "row", alignItems: "center", gap: 6 },
  block: { borderWidth: 1, borderRadius: 20, padding: 16, gap: 8 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  actions: { gap: 10, marginTop: 8 }
});
