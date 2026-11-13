import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { TabScreenProps } from "../../navigation/types";
import { useMyProfile } from "../../state/queries";
import { useTheme } from "../../theme/ThemeProvider";
import { Divider, ErrorState, ListRow, LoadingState, Photo, ProgressBar, Screen, Text } from "../../ui";
import { describeError } from "../../utils/errors";

const TOTAL_REQUIRED = 6;

export default function MeScreen({ navigation }: TabScreenProps<"Me">) {
  const { colors, spacing, radii } = useTheme();
  const profile = useMyProfile();

  if (profile.isLoading) {
    return (
      <Screen edges={["top"]}>
        <LoadingState />
      </Screen>
    );
  }
  if (!profile.data) {
    return (
      <Screen edges={["top"]}>
        <ErrorState message={describeError(profile.error)} onAction={() => void profile.refetch()} />
      </Screen>
    );
  }
  const p = profile.data;
  const completion = (TOTAL_REQUIRED - p.missing.length) / TOTAL_REQUIRED;

  return (
    <Screen scroll edges={["top"]} testID="me-screen">
      <View style={{ gap: spacing.lg, paddingTop: spacing.sm }}>
        <Text variant="title" accessibilityRole="header">
          Mon profil
        </Text>
        <Pressable
          onPress={() => navigation.navigate("MyPreview")}
          accessibilityRole="button"
          accessibilityLabel="Voir mon profil comme les autres"
          style={[styles.card, { backgroundColor: colors.surface, borderRadius: radii.xl, borderColor: colors.border }]}
        >
          <Photo url={p.photos[0]?.url} style={[styles.photo, { borderRadius: radii.lg }]} />
          <View style={styles.flex}>
            <Text variant="heading" testID="me-name">
              {p.firstName || "Votre prénom"}
              {p.age ? `, ${p.age}` : ""}
            </Text>
            <Text tone="muted" variant="label">
              {p.city || "Ville non renseignée"}
            </Text>
            <Text tone="primary" variant="label" weight="semibold" style={{ marginTop: 6 }}>
              Voir l’aperçu
            </Text>
          </View>
        </Pressable>

        {!p.isComplete ? (
          <Pressable
            onPress={() => navigation.navigate("CompleteProfile")}
            style={[styles.notice, { backgroundColor: colors.primarySoft, borderRadius: radii.lg }]}
            accessibilityRole="button"
          >
            <Text weight="semibold">Profil complété à {Math.round(completion * 100)} %</Text>
            <ProgressBar value={completion} />
            <Text variant="label" tone="muted">
              Complétez-le pour apparaître dans la découverte.
            </Text>
          </Pressable>
        ) : p.isPaused ? (
          <View style={[styles.notice, { backgroundColor: colors.accentSoft, borderRadius: radii.lg }]}>
            <Text weight="semibold" tone="accent">
              Profil en pause
            </Text>
            <Text variant="label" tone="muted">
              Vous restez invisible dans la découverte. Vos conversations continuent.
            </Text>
          </View>
        ) : null}

        <View
          style={[
            styles.group,
            { backgroundColor: colors.surface, borderRadius: radii.lg, borderColor: colors.border }
          ]}
        >
          <ListRow
            icon="create-outline"
            label="Modifier mon profil"
            description="Photos, bio, centres d'intérêt"
            onPress={() => navigation.navigate("EditProfile")}
            testID="me-edit"
          />
          <Divider />
          <ListRow
            icon="options-outline"
            label="Préférences de rencontre"
            description="Genre, âge, distance"
            onPress={() => navigation.navigate("Preferences")}
            testID="me-preferences"
          />
          <Divider />
          <ListRow
            icon="shield-checkmark-outline"
            label="Confidentialité et sécurité"
            description="Pause, distance, blocages, compte"
            onPress={() => navigation.navigate("Settings")}
            testID="me-settings"
          />
        </View>
        <Text variant="caption" tone="subtle" align="center">
          Aimant est entièrement gratuit : pas d’abonnement, pas de publicité, pas de limite de likes.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  card: { flexDirection: "row", gap: 16, padding: 12, alignItems: "center", borderWidth: StyleSheet.hairlineWidth },
  photo: { width: 84, height: 105 },
  notice: { padding: 16, gap: 8 },
  group: { paddingHorizontal: 14, borderWidth: StyleSheet.hairlineWidth }
});
