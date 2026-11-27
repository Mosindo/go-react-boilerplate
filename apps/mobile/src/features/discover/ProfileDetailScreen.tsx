import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { GENDER_LABELS } from "../../api/types";
import { useMyProfile, usePublicProfile } from "../../hooks/queries";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, spacing } from "../../theme/palette";
import { Button, Chip, ErrorState, Header, Icon, Loading, RemoteImage, Screen, Section, Text } from "../../ui";
import { formatDistance } from "../../utils/format";
import type { AppScreenProps } from "../../navigation/types";
import { ReportSheet } from "../safety/ReportSheet";
import { useSafetyActions } from "../safety/useSafetyActions";

export function ProfileDetailScreen({ navigation, route }: AppScreenProps<"ProfileDetail">) {
  const { userId } = route.params;
  const { colors } = useTheme();
  const profile = usePublicProfile(userId);
  const me = useMyProfile();
  const [reporting, setReporting] = useState(false);
  const safety = useSafetyActions(() => {
    setReporting(false);
    navigation.popToTop();
  });

  if (profile.isLoading) {
    return (
      <Screen>
        <Header onBack={() => navigation.goBack()} />
        <Loading />
      </Screen>
    );
  }
  if (profile.error || !profile.data) {
    return (
      <Screen>
        <Header onBack={() => navigation.goBack()} />
        <ErrorState message={profile.error ? errorMessage(profile.error) : "Profil introuvable."} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }

  const card = profile.data;
  const mine = new Set((me.data?.interests ?? []).map((i) => i.id));
  const distance = formatDistance(card.distanceKm);

  return (
    <Screen scroll testID="profile-detail">
      <Header title={card.firstName} onBack={() => navigation.goBack()} />
      <View style={styles.content}>
        {card.photos[0] ? (
          <RemoteImage url={card.photos[0].url} style={styles.photo} priority="high" accessibilityLabel={`Photo 1 de ${card.firstName}`} />
        ) : null}
        <View style={styles.identity}>
          <Text variant="title">
            {card.firstName}, {card.age}
          </Text>
          <View style={styles.meta}>
            <Icon name="person-outline" size={15} color={colors.textMuted} />
            <Text variant="label" tone="muted">
              {GENDER_LABELS[card.gender]}
            </Text>
            {distance || card.city ? (
              <>
                <Icon name="location-outline" size={15} color={colors.textMuted} />
                <Text variant="label" tone="muted">
                  {[distance, card.city].filter(Boolean).join(" · ")}
                </Text>
              </>
            ) : null}
          </View>
          {card.isDemo ? (
            <Text variant="caption" tone="accent">
              Profil de démonstration (données fictives)
            </Text>
          ) : null}
        </View>

        {card.bio ? (
          <Section title="À propos">
            <Text>{card.bio}</Text>
          </Section>
        ) : null}

        {card.interests.length ? (
          <Section title="Centres d'intérêt">
            <View style={styles.chips}>
              {card.interests.map((i) => (
                <Chip key={i.id} label={i.label} highlighted={mine.has(i.id)} icon={mine.has(i.id) ? "sparkles" : undefined} />
              ))}
            </View>
          </Section>
        ) : null}

        {card.photos.slice(1).map((photo, index) => (
          <RemoteImage key={photo.id} url={photo.url} style={styles.photo} priority="low" accessibilityLabel={`Photo ${index + 2} de ${card.firstName}`} />
        ))}

        <Section title="Sécurité">
          <View style={styles.safety}>
            <Button label="Signaler" variant="outline" icon="flag-outline" onPress={() => setReporting(true)} style={styles.flex} testID="profile-report" />
            <Button
              label="Bloquer"
              variant="danger"
              icon="ban-outline"
              loading={safety.block.isPending}
              onPress={() => void safety.confirmBlock(card.userId, card.firstName)}
              style={styles.flex}
              testID="profile-block"
            />
          </View>
        </Section>
      </View>
      <ReportSheet
        visible={reporting}
        name={card.firstName}
        loading={safety.report.isPending}
        onClose={() => setReporting(false)}
        onSubmit={(input) => safety.report.mutate({ userId: card.userId, ...input })}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { gap: spacing.md, maxWidth: 560, width: "100%", alignSelf: "center" },
  photo: { width: "100%", aspectRatio: 3 / 4, borderRadius: radii.lg },
  identity: { gap: 4 },
  meta: { flexDirection: "row", alignItems: "center", gap: 4, flexWrap: "wrap" },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  safety: { flexDirection: "row", gap: spacing.sm },
  flex: { flex: 1 }
});
