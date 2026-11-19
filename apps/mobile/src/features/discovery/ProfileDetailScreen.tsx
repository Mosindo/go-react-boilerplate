import React, { useState } from "react";
import { ScrollView, StyleSheet, useWindowDimensions, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { SafeAreaView } from "react-native-safe-area-context";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import { formatDistance, genderLabels, lookingForLabels } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, spacing } from "../../theme/tokens";
import { ActionSheet, Chip, ErrorState, Icon, IconButton, LoadingState, RemoteImage, Text, useFeedback } from "../../ui";
import { ReportSheet } from "../safety/ReportSheet";
import { useBlock } from "../safety/useBlock";
import { useSwipe } from "./useSwipe";

type Props = NativeStackScreenProps<AppStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, canSwipe } = route.params;
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const { toast } = useFeedback();
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);
  const swipe = useSwipe();
  const block = useBlock(() => navigation.goBack());
  const query = useQuery({ queryKey: queryKeys.publicProfile(userId), queryFn: () => profileApi.view(userId) });

  const card = query.data;
  const photoWidth = Math.min(width, 640);

  const decide = (decision: "like" | "pass") =>
    swipe.mutate(
      { userId, decision },
      {
        onSuccess: () => navigation.goBack(),
        onError: (e) => toast(errorMessage(e), "error")
      }
    );

  return (
    <SafeAreaView style={[styles.root, { backgroundColor: colors.background }]} edges={["top", "left", "right"]}>
      <View style={styles.topBar}>
        <IconButton icon="chevron-down" label="Fermer" tone="default" onPress={navigation.goBack} />
        {card ? (
          <IconButton
            icon="ellipsis-horizontal"
            label="Plus d'options"
            tone="default"
            onPress={() => setMenu(true)}
            testID="profile-menu"
          />
        ) : null}
      </View>
      {query.isLoading ? (
        <LoadingState />
      ) : query.error || !card ? (
        <ErrorState message={query.error ? errorMessage(query.error) : "Profil indisponible."} onRetry={() => void query.refetch()} />
      ) : (
        <>
          <ScrollView contentContainerStyle={styles.scroll}>
            <View style={[styles.inner, { maxWidth: 640 }]}>
              {card.photos.map((p, i) => (
                <React.Fragment key={p.id}>
                  <RemoteImage
                    path={p.url}
                    style={[styles.photo, { width: photoWidth - spacing.lg * 2, height: (photoWidth - spacing.lg * 2) * 1.25 }]}
                    accessibilityLabel={`Photo ${i + 1} de ${card.firstName}`}
                  />
                  {i === 0 ? (
                    <View style={styles.section}>
                      <Text variant="display">
                        {card.firstName}, {card.age}
                      </Text>
                      <View style={styles.facts}>
                        {card.jobTitle ? <Fact icon="briefcase-outline" label={card.jobTitle} /> : null}
                        {formatDistance(card.distanceKm) ? (
                          <Fact icon="location-outline" label={formatDistance(card.distanceKm) ?? ""} />
                        ) : null}
                        {card.city ? <Fact icon="home-outline" label={card.city} /> : null}
                        <Fact icon="person-outline" label={genderLabels[card.gender]} />
                        <Fact icon="search-outline" label={lookingForLabels[card.lookingFor]} />
                      </View>
                      {card.bio ? <Text style={styles.bio}>{card.bio}</Text> : null}
                      {card.interests.length ? (
                        <View style={styles.chips}>
                          {card.interests.map((i) => (
                            <Chip key={i.id} label={i.label} />
                          ))}
                        </View>
                      ) : null}
                    </View>
                  ) : null}
                </React.Fragment>
              ))}
            </View>
          </ScrollView>
          {canSwipe ? (
            <View style={styles.actions}>
              <IconButton icon="close" label="Passer" tone="default" size={60} disabled={swipe.isPending} onPress={() => decide("pass")} />
              <IconButton
                icon="heart"
                label="J'aime"
                tone="primary"
                size={68}
                disabled={swipe.isPending}
                onPress={() => decide("like")}
                testID="detail-like"
              />
            </View>
          ) : null}
          <ActionSheet
            visible={menu}
            title={card.firstName}
            onClose={() => setMenu(false)}
            actions={[
              {
                label: `Signaler ${card.firstName}`,
                icon: "flag-outline",
                destructive: true,
                onPress: () => setReporting(true),
                testID: "menu-report"
              },
              {
                label: `Bloquer ${card.firstName}`,
                icon: "ban-outline",
                destructive: true,
                onPress: () => void block(userId, card.firstName),
                testID: "menu-block"
              }
            ]}
          />
          <ReportSheet
            userId={userId}
            name={card.firstName}
            visible={reporting}
            onClose={() => setReporting(false)}
            onDone={() => {
              setReporting(false);
              navigation.goBack();
            }}
          />
        </>
      )}
    </SafeAreaView>
  );
}

function Fact({ icon, label }: { icon: React.ComponentProps<typeof Icon>["name"]; label: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.fact}>
      <Icon name={icon} size={16} color={colors.textMuted} />
      <Text tone="muted">{label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  topBar: { flexDirection: "row", justifyContent: "space-between", paddingHorizontal: spacing.lg, paddingVertical: spacing.xs },
  scroll: { paddingBottom: 120 },
  inner: { width: "100%", alignSelf: "center", paddingHorizontal: spacing.lg, gap: spacing.lg },
  photo: { borderRadius: radii.lg },
  section: { gap: spacing.sm },
  facts: { gap: 6 },
  fact: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  bio: { marginTop: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs, marginTop: spacing.xs },
  actions: {
    position: "absolute",
    bottom: spacing.xl,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xl
  }
});
